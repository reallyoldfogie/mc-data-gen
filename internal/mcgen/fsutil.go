package mcgen

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
)

// CopyDir recursively copies a directory tree.
func CopyDir(src, dst string) error {
    info, err := os.Stat(src)
    if err != nil {
        return fmt.Errorf("stat src: %w", err)
    }
    if err := os.MkdirAll(dst, info.Mode()); err != nil {
        return fmt.Errorf("mkdir dst: %w", err)
    }

    entries, err := os.ReadDir(src)
    if err != nil {
        return fmt.Errorf("read dir: %w", err)
    }

    for _, e := range entries {
        srcPath := filepath.Join(src, e.Name())
        dstPath := filepath.Join(dst, e.Name())

        if e.IsDir() {
            if err := CopyDir(srcPath, dstPath); err != nil {
                return err
            }
            continue
        }

        if err := copyFile(srcPath, dstPath); err != nil {
            return err
        }
    }
    return nil
}

func copyFile(src, dst string) error {
    in, err := os.Open(src)
    if err != nil {
        return fmt.Errorf("open src: %w", err)
    }
    defer in.Close()

    info, err := in.Stat()
    if err != nil {
        return fmt.Errorf("stat src: %w", err)
    }

    out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
    if err != nil {
        return fmt.Errorf("create dst: %w", err)
    }
    defer out.Close()

    if _, err := io.Copy(out, in); err != nil {
        return fmt.Errorf("copy: %w", err)
    }
    return nil
}
