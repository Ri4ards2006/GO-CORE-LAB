// ═══════════════════════════════════════════════════════════════════════════
// Package mmap provides a high-performance, safe memory-mapping abstraction
// for zero-copy file inspection on Linux and POSIX systems.
// ═══════════════════════════════════════════════════
package mmap

import (
	"fmt"
	"os"
	"syscall"
)

// File wraps a memory-mapped byte slice backed by a physical file descriptor.
type File struct {
	data     []byte
	file     *os.File
	isMapped bool
}

// Open maps the named file into the process virtual address space as read-only.
func Open(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file %q: %w", path, err)
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("stat file %q: %w", path, err)
	}

	size := fi.Size()
	if size <= 0 {
		// Zero-byte file: return empty slice without syscall.Mmap
		return &File{
			data:     []byte{},
			file:     f,
			isMapped: false,
		}, nil
	}

	// Attempt memory mapping with PROT_READ and MAP_SHARED
	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		// Fallback to buffered read if mmap fails (e.g. unsupported filesystem or virtual device)
		buf, readErr := os.ReadFile(path)
		if readErr != nil {
			f.Close()
			return nil, fmt.Errorf("mmap failed (%v) and fallback read failed: %w", err, readErr)
		}
		return &File{
			data:     buf,
			file:     f,
			isMapped: false,
		}, nil
	}

	return &File{
		data:     data,
		file:     f,
		isMapped: true,
	}, nil
}

// Bytes returns the direct memory-mapped byte slice without allocation.
func (f *File) Bytes() []byte {
	return f.data
}

// Len returns the total size in bytes of the memory-mapped file.
func (f *File) Len() int {
	return len(f.data)
}

// Slice returns a sub-slice within bounds without heap copying.
func (f *File) Slice(offset, length int) ([]byte, error) {
	if offset < 0 || length < 0 || offset+length > len(f.data) {
		return nil, fmt.Errorf("slice [0x%x : 0x%x] out of bounds (file len: 0x%x)",
			offset, offset+length, len(f.data))
	}
	return f.data[offset : offset+length], nil
}

// IsMapped returns true if the backing storage is actively memory-mapped.
func (f *File) IsMapped() bool {
	return f.isMapped
}

// Close unmaps the memory region and closes the underlying file descriptor.
func (f *File) Close() error {
	var mmapErr error
	if f.isMapped && len(f.data) > 0 {
		mmapErr = syscall.Munmap(f.data)
		f.isMapped = false
		f.data = nil
	}

	var fileErr error
	if f.file != nil {
		fileErr = f.file.Close()
		f.file = nil
	}

	if mmapErr != nil {
		return fmt.Errorf("munmap: %w", mmapErr)
	}
	return fileErr
}

