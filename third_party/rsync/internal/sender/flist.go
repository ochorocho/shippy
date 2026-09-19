package sender

import (
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gokrazy/rsync"
	"github.com/gokrazy/rsync/internal/rsyncchecksum"
	"github.com/gokrazy/rsync/internal/rsyncopts"
	"github.com/gokrazy/rsync/internal/rsyncwire"
)

type file struct {
	source  FileSource
	path    string
	Wpath   string
	regular bool

	// fields below are used by the receiver (TODO: unify)
	Name       string
	Length     int64
	ModTime    time.Time
	Mode       int32
	Uid        int32
	Gid        int32
	LinkTarget string
	Rdev       int32
}

type fileList struct {
	TotalSize int64
	Files     []file
	Sources   []FileSource
}

// A fileList must not be used after calling Close().
func (fl *fileList) Close() {
	for _, source := range fl.Sources {
		source.Close()
	}
	fl.Sources = nil
}

// rsync/rsync.h defines chunkSize as 32 * 1024, which we match.
//
// Other parts of rsync have subtle dependencies on the chunkSize.
// For example, (*sender.Transfer).hashSearch declares a window
// of at least 256 KB and then reads from that window, requesting
// 2*chunkSize+alignFudge, which must not exceed the window.
// So raising chunkSize to 128 KB or 256 KB breaks transfers
// with "file has changed mid-transfer" (issue #53).
const chunkSize = 32 * 1024

var (
	lookupOnce      sync.Once
	lookupGroupOnce sync.Once
)

// getStrip operates on wire paths (slash space).
func getStrip(requested string) string {
	if requested == "/" {
		return ""
	}
	if strings.HasSuffix(requested, "/") {
		return strings.TrimPrefix(path.Clean(requested), "/") + "/"
	}
	return ""
}

type scopedWalker struct {
	st        *Transfer
	ioError   func(err error)
	conn      *rsyncwire.Conn
	fec       *rsyncwire.Buffer
	excl      *filterRuleList
	uidMap    map[int32]string
	gidMap    map[int32]string
	fileList  *fileList
	source    FileSource
	localDir  string
	requested string
	strip     string
	subdir    string
	prefix    string
}

// openSource resolves s.source from s.localDir (and s.subdir). Shared by walk()
// and the --files-from builder (SHIPPY PATCH).
func (s *scopedWalker) openSource() error {
	if s.source == nil {
		fi, err := os.Lstat(s.localDir)
		if err != nil {
			s.st.Logger.Printf("  Lstat(localDir=%q): %v", s.localDir, err)
			return fmt.Errorf("i/o error: requested module path is not accessible")
		}
		if fi.IsDir() {
			root, err := os.OpenRoot(s.localDir)
			if err != nil {
				s.st.Logger.Printf("  OpenRoot(localDir=%q): %v", s.localDir, err)
				return fmt.Errorf("i/o error: requested module path is not accessible")
			}
			s.source = newOSRootSource(root)
		} else {
			s.source = newSingleFileSource(s.localDir, fi)
		}
		s.fileList.Sources = append(s.fileList.Sources, s.source)
	}
	if s.subdir != "." {
		sub, err := newSubSource(s.source, s.subdir)
		if err != nil {
			s.st.Logger.Printf("  newSubSource(subdir=%q): %v", s.subdir, err)
			return fmt.Errorf("i/o error: requested module path is not accessible")
		}
		s.source = sub
	}
	return nil
}

func (s *scopedWalker) walk() error {
	if err := s.openSource(); err != nil {
		return err
	}

	rootname := s.requested
	// fs.WalkDir(root.FS(), …) does not accept absolute paths,
	// so make them relative by prepending a .
	if strings.HasPrefix(rootname, "/") {
		rootname = "." + rootname
	}
	if err := fs.WalkDir(s.source.FS(), path.Clean(rootname), s.walkFn); err != nil {
		return err
	}
	return nil
}

func (s *scopedWalker) walkFn(path string, d fs.DirEntry, err error) error {
	logger := s.st.Logger // for convenience
	opts := s.st.Opts     // for convenience
	if opts.DebugGTE(rsyncopts.DEBUG_FLIST, 1) {
		logger.Printf("filepath.WalkFn(path=%s)", path)
	}
	var info fs.FileInfo
	if err == nil {
		info, err = d.Info()
	}
	if err != nil {
		// set the I/O error flag, but keep walking
		s.ioError(err)
		return nil
	}

	if opts.DebugGTE(rsyncopts.DEBUG_FLIST, 1) {
		logger.Printf("isDir=%v, xferDirs=%v", info.Mode().IsDir(), opts.XferDirs())
	}
	if info.Mode().IsDir() && opts.XferDirs() == 0 {
		logger.Printf("skipping directory %s", path)
		return filepath.SkipDir
	}

	// Only ever transmit long names, like openrsync
	flags := byte(rsync.XMIT_LONG_NAME)

	name := path
	if s.strip != "" {
		if path+"/" == s.strip {
			// Transmit the top directory as ., like tridge rsync.
			name = "."
		} else {
			name = strings.TrimPrefix(name, s.strip)
		}
	}
	if s.prefix != "" {
		if path == "." {
			name = s.prefix
		} else {
			name = s.prefix + "/" + path
		}
	}
	if opts.DebugGTE(rsyncopts.DEBUG_FLIST, 1) {
		logger.Printf("Trim(path=%q) = %q", path, name)
	}
	if path == "." && s.prefix == "" {
		flags |= rsync.XMIT_TOP_DIR
	}
	// st.logger.Printf("flags for %q: %v", name, flags)

	if s.excl.matches(name) {
		return filepath.SkipDir
	}

	if err := s.emit(path, name, flags, info); err != nil {
		return err
	}

	if info.Mode().IsDir() && !opts.Recurse() {
		return filepath.SkipDir
	}

	return nil
}

// emit appends one entry to the file list and encodes it onto the wire. It is
// shared by the recursive walker (walkFn) and the --files-from builder
// (emitList). path is the entry's path within the source FS (used for
// Readlink/Open); name is its wire path (Wpath); flags is the pre-computed
// status byte.
//
// SHIPPY PATCH: factored out of walkFn so --files-from can reuse it. See
// SHIPPY_PATCHES.md.
func (s *scopedWalker) emit(path, name string, flags byte, info fs.FileInfo) error {
	logger := s.st.Logger
	opts := s.st.Opts

	s.fileList.Files = append(s.fileList.Files, file{
		source:  s.source,
		path:    path,
		regular: info.Mode().IsRegular(),
		Wpath:   name,
		Length:  info.Size(),
	})

	s.fec.Reset()

	// 1.   status byte (integer)
	s.fec.WriteByte(flags)

	// 2.   inherited filename length (optional, byte)
	// 3.   filename length (integer or byte)
	s.fec.WriteInt32(int32(len(name)))

	// 4.   file (byte array)
	s.fec.WriteString(name)

	// 5.   file length (long)
	size := info.Size()
	if info.Mode().IsDir() {
		// tmpfs returns non-4K sizes for directories. Override with
		// 4096 to make the tests succeed regardless of the /tmp file
		// system type.
		size = 4096
	}
	s.fec.WriteInt64(size)

	s.fileList.TotalSize += size

	// 6.   file modification time (optional, integer)
	// TODO: this will overflow in 2038! :(
	s.fec.WriteInt32(int32(info.ModTime().Unix()))

	// 7.   file mode (optional, mode_t, integer)
	mode := int32(info.Mode() & os.ModePerm)
	isDev := false
	isSpecial := false
	if info.Mode().IsDir() {
		mode |= rsync.S_IFDIR
	} else if info.Mode().IsRegular() {
		mode |= rsync.S_IFREG
	} else if info.Mode().Type()&os.ModeSymlink != 0 {
		mode |= rsync.S_IFLNK
		// TODO: skip symlink if PreserveSymlinks is not set
	}

	if info.Mode().Type()&os.ModeCharDevice != 0 {
		mode |= rsync.S_IFCHR
		isDev = true
	} else if info.Mode().Type()&os.ModeDevice != 0 {
		mode |= rsync.S_IFBLK
		isDev = true
	}

	if info.Mode().Type()&os.ModeNamedPipe != 0 {
		mode |= rsync.S_IFIFO
		isSpecial = true
	}

	if info.Mode().Type()&os.ModeSocket != 0 {
		mode |= rsync.S_IFSOCK
		isSpecial = true
	}

	s.fec.WriteInt32(mode)

	if opts.PreserveUid() {
		uid, ok := uidFromFileInfo(info)
		if ok {
			if _, ok := s.uidMap[uid]; !ok && uid != 0 {
				u, err := user.LookupId(strconv.Itoa(int(uid)))
				if err != nil {
					lookupOnce.Do(func() {
						logger.Printf("lookup(%d) = %v", uid, err)
					})
				} else {
					s.uidMap[uid] = u.Username
				}
			}
		}
		// 8.   if -o, the user id (integer)
		s.fec.WriteInt32(uid)
	}

	if opts.PreserveGid() {
		gid, ok := gidFromFileInfo(info)
		if ok {
			if _, ok := s.gidMap[gid]; !ok && gid != 0 {
				g, err := user.LookupGroupId(strconv.Itoa(int(gid)))
				if err != nil {
					lookupGroupOnce.Do(func() {
						logger.Printf("lookupgroup(%d) = %v", gid, err)
					})
				} else {
					s.gidMap[gid] = g.Name
				}
			}
		}
		// 9.   if -g, the group id (integer)
		s.fec.WriteInt32(gid)
	}

	if (opts.PreserveDevices() && isDev) ||
		(opts.PreserveSpecials() && isSpecial) {
		// 10.  if a special file and -D, the device “rdev” type (integer)
		rdev, _ := rdevFromFileInfo(info)
		s.fec.WriteInt32(rdev)
	}

	if opts.PreserveLinks() && info.Mode().Type()&os.ModeSymlink != 0 {
		// 11.  if a symbolic link and -l, the link target's length (integer)
		// 12.  if a symbolic link and -l, the link target (byte array)

		target, err := s.source.Readlink(path)
		if err != nil {
			return err // TODO
		}
		s.fec.WriteInt32(int32(len(target)))
		s.fec.WriteString(target)
	}

	if opts.AlwaysChecksum() {
		var emptyChecksum [rsyncchecksum.Size]byte
		checksum := emptyChecksum[:]
		if info.Mode().IsRegular() {
			f, err := s.source.Open(path)
			if err != nil {
				return err
			}
			checksum, err = rsyncchecksum.ReaderChecksum(f)
			f.Close()
			if err != nil {
				return err
			}
		} else {
			// send empty md4 checksum
		}
		s.fec.WriteString(string(checksum))
	}

	s.conn.WriteString(s.fec.String())

	// The status byte may consist of the following bits and determines which of the optional fields are transmitted.

	// 0x01    A top-level directory.  (Only applies to directory files.)  If specified, the matching local directory is for deletions.
	// 0x02    Do not send the file mode: it is a repeat of the last file's mode.
	// 0x08    Like 0x02, but for the user id.
	// 0x10    Like 0x02, but for the group id.
	// 0x20    Inherit some of the prior file name.  Enables the inherited filename length transmission.
	// 0x40    Use full integer length for file name.  Otherwise, use only the byte length.
	// 0x80    Do not send the file modification time: it is a repeat of the last file's.

	// If the status byte is zero, the file-list has terminated.

	return nil
}

// emitList sends an explicit list of relative paths (--files-from), synthesizing
// the implied parent directories rsync needs to recreate the tree, then emitting
// each entry via the shared emit(). SHIPPY PATCH: see SHIPPY_PATCHES.md.
func (s *scopedWalker) emitList(list []string) error {
	if err := s.openSource(); err != nil {
		return err
	}

	// Build the entry set: every listed path plus each of its ancestor
	// directories, deduplicated. Lexical sort yields root-before-leaf order.
	seen := make(map[string]bool)
	var entries []string
	add := func(p string) {
		if p == "" || p == "." || seen[p] {
			return
		}
		seen[p] = true
		entries = append(entries, p)
	}
	for _, rel := range list {
		rel = path.Clean(strings.TrimPrefix(rel, "/"))
		if rel == "." || rel == "" {
			continue
		}
		var dirs []string
		for d := path.Dir(rel); d != "." && d != "/"; d = path.Dir(d) {
			dirs = append(dirs, d)
		}
		for i := len(dirs) - 1; i >= 0; i-- {
			add(dirs[i])
		}
		add(rel)
	}
	sort.Strings(entries)

	// SHIPPY PATCH: emit the root "." as a top-level directory so a --delete
	// receiver treats the whole tree as a deletion scope and prunes recursively
	// (files whose entire parent directory was removed are otherwise missed).
	if rootInfo, err := fs.Lstat(s.source.FS(), "."); err == nil {
		rootWpath := "."
		if s.prefix != "" {
			rootWpath = s.prefix
		}
		if err := s.emit(".", rootWpath, byte(rsync.XMIT_LONG_NAME|rsync.XMIT_TOP_DIR), rootInfo); err != nil {
			return err
		}
	}

	for _, name := range entries {
		info, err := fs.Lstat(s.source.FS(), name)
		if err != nil {
			s.ioError(err)
			continue
		}
		wpath := name
		if s.prefix != "" {
			wpath = s.prefix + "/" + name
		}
		flags := byte(rsync.XMIT_LONG_NAME)
		// Mark top-level directories so the receiver treats them as deletion
		// scopes (FLAG_TOP_DIR). Only applies to directories.
		if info.IsDir() && !strings.Contains(name, "/") {
			flags |= rsync.XMIT_TOP_DIR
		}
		if err := s.emit(name, wpath, flags, info); err != nil {
			return err
		}
	}
	return nil
}

// readFilesFrom reads and splits a --files-from list file. SHIPPY PATCH.
func readFilesFrom(pathname string, nulSep bool) ([]string, error) {
	data, err := os.ReadFile(pathname)
	if err != nil {
		return nil, fmt.Errorf("--files-from: %w", err)
	}
	sep := "\n"
	if nulSep {
		sep = "\x00"
	}
	var list []string
	for _, line := range strings.Split(string(data), sep) {
		if !nulSep {
			line = strings.TrimRight(line, "\r")
		}
		if line == "" {
			continue
		}
		list = append(list, line)
	}
	return list, nil
}

// rsync/flist.c:send_file_list
func (st *Transfer) SendFileList(localDir string, paths []string, excl *filterRuleList) (*fileList, error) {
	var fileList fileList
	fec := &rsyncwire.Buffer{}

	uidMap := make(map[int32]string)
	gidMap := make(map[int32]string)

	// TODO: flush in between to keep the pipes filled when traversal takes long

	// TODO: handle info == nil case (permission denied?): should set an i/o
	// error flag, but traversal should continue

	st.Logger.Printf("building file list")
	if st.Opts.DebugGTE(rsyncopts.DEBUG_FLIST, 1) {
		st.Logger.Printf("sendFileList()")
	}
	ioErrors := int32(0)

	ioError := func(err error) {
		if os.IsNotExist(err) {
			st.Logger.Printf("file vanished: %v", err)
		} else {
			st.Logger.Printf("lstat: %v", err)
		}
		ioErrors = 1
	}

	// SHIPPY PATCH: --files-from drives the transfer from an explicit list
	// instead of walking the source tree. See SHIPPY_PATCHES.md.
	if from := st.Opts.FilesFrom(); from != "" {
		list, err := readFilesFrom(from, st.Opts.EolNulls())
		if err != nil {
			return nil, err
		}
		if len(paths) != 1 {
			return nil, fmt.Errorf("--files-from requires exactly one source path, got %d", len(paths))
		}
		requested := paths[0]
		local := localDir
		prefix := ""
		if local == rsync.FileSystemRoot {
			local = filepath.Clean(requested)
			if !strings.HasSuffix(requested, "/") {
				prefix = filepath.Base(requested)
			}
			requested = "/"
		}
		sw := &scopedWalker{
			st:        st,
			conn:      st.Conn,
			fec:       fec,
			excl:      excl,
			uidMap:    uidMap,
			gidMap:    gidMap,
			fileList:  &fileList,
			source:    st.Source,
			ioError:   ioError,
			localDir:  local,
			requested: requested,
			strip:     getStrip(requested),
			subdir:    ".",
			prefix:    prefix,
		}
		if err := sw.emitList(list); err != nil {
			return nil, err
		}
	} else {
		for _, requested := range paths {
			subdir := "."
			prefix := ""
			local := localDir
			if local == rsync.FileSystemRoot {
				// Implicit module (/) and absolute requested path (/tmp/foo/),
				// turn the path into the local directory and request /.
				local = filepath.Clean(requested)
				if strings.HasSuffix(requested, "/") {

				} else {
					prefix = filepath.Base(requested)
				}
				requested = "/"
			} else if !strings.HasSuffix(requested, "/") {
				st.Logger.Printf("  handling requested=%q", requested)
				clean := path.Clean(strings.TrimPrefix(requested, "/"))
				subdir = path.Dir(clean)
				requested = path.Base(clean)
				st.Logger.Printf("  -> subdir=%q, requested=%q", subdir, requested)
			}

			if st.Opts.DebugGTE(rsyncopts.DEBUG_FLIST, 1) {
				st.Logger.Printf("  path %q (local dir %q)", requested, local)
			}
			// st.Logger.Printf("getRootStrip(requested=%q, localDir=%q", requested, localDir)
			strip := getStrip(requested)
			// st.Logger.Printf("root=%q, strip=%q", root, strip)
			if st.Opts.DebugGTE(rsyncopts.DEBUG_FLIST, 1) {
				st.Logger.Printf("  fs.Walk(%q, %q), strip=%q", local, requested)
			}

			sw := &scopedWalker{
				st:        st,
				conn:      st.Conn,
				fec:       fec,
				excl:      excl,
				uidMap:    uidMap,
				gidMap:    gidMap,
				fileList:  &fileList,
				source:    st.Source,
				ioError:   ioError,
				localDir:  local,
				requested: requested,
				strip:     strip,
				subdir:    subdir,
				prefix:    prefix,
			}
			if err := sw.walk(); err != nil {
				return nil, err
			}
		}
	}

	if st.Opts.InfoGTE(rsyncopts.INFO_PROGRESS, 1) {
		st.Logger.Printf("%d files to consider", len(fileList.Files))
	}

	fec.Reset()

	const endOfFileList = 0
	fec.WriteByte(endOfFileList)

	const endOfSet = 0
	if st.Opts.PreserveUid() {
		for uid, name := range uidMap {
			fec.WriteInt32(uid)
			fec.WriteByte(byte(len(name)))
			fec.WriteString(name)
		}
		fec.WriteInt32(endOfSet)
	}
	if st.Opts.PreserveGid() {
		for gid, name := range gidMap {
			fec.WriteInt32(gid)
			fec.WriteByte(byte(len(name)))
			fec.WriteString(name)
		}
		fec.WriteInt32(endOfSet)
	}

	fec.WriteInt32(ioErrors)

	if err := st.Conn.WriteString(fec.String()); err != nil {
		return nil, err
	}

	return &fileList, nil
}
