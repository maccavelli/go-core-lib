//go:build windows

package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

type cleanupReceipt struct {
	Version int    `json:"version"`
	Backup  string `json:"backup"`
	Digest  string `json:"digest"`
}

func fileSHA256(path string) (digest string, err error) {
	f, err := openAbsFile(path, os.O_RDONLY, 0)
	if err != nil {
		return "", err
	}
	defer func() {
		err = joinClose(err, f)
	}()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func processCleanupReceipt(target Target, root *os.Root) error {
	name := cleanupReceiptName(target.Base)
	info, err := root.Lstat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("selfupdate: stat cleanup receipt: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("selfupdate: cleanup receipt is not a regular file")
	}
	if isReparsePoint(filepath.Join(target.Dir, name)) {
		return fmt.Errorf("selfupdate: cleanup receipt is a reparse point")
	}
	data, err := root.ReadFile(name)
	if err != nil {
		return fmt.Errorf("selfupdate: read cleanup receipt: %w", err)
	}
	var rec cleanupReceipt
	if err := json.Unmarshal(data, &rec); err != nil {
		return fmt.Errorf("selfupdate: malformed cleanup receipt: %w", err)
	}
	if rec.Version != 1 || rec.Backup == "" || rec.Digest == "" {
		return fmt.Errorf("selfupdate: malformed cleanup receipt")
	}
	if err := validateReceiptBackup(target, rec.Backup); err != nil {
		return err
	}
	backupPath := filepath.Join(target.Dir, rec.Backup)
	binfo, err := root.Lstat(rec.Backup)
	if errors.Is(err, os.ErrNotExist) {
		// The backup is already gone (a crash between the two removals, or
		// a manual delete): the receipt has nothing left to protect, and
		// keeping it would block every later update (0003-MADR B7).
		if err := root.Remove(name); err != nil {
			return fmt.Errorf("selfupdate: remove stale cleanup receipt: %w", err)
		}
		return syncDirFn(target.Dir)
	}
	if err != nil {
		return fmt.Errorf("selfupdate: stat pending backup: %w", err)
	}
	if !binfo.Mode().IsRegular() || isReparsePoint(backupPath) {
		return fmt.Errorf("selfupdate: pending backup is not a regular file")
	}
	got, err := fileSHA256(backupPath)
	if err != nil {
		return err
	}
	if got != rec.Digest {
		return fmt.Errorf("selfupdate: pending backup digest mismatch: %w", ErrIntegrity)
	}
	if err := root.Remove(rec.Backup); err != nil {
		return fmt.Errorf("selfupdate: remove pending backup: %w", err)
	}
	if err := root.Remove(name); err != nil {
		return fmt.Errorf("selfupdate: remove cleanup receipt: %w", err)
	}
	return syncDirFn(target.Dir)
}

func writeCleanupReceipt(target Target, result applyResult) error {
	base := filepath.Base(result.backup)
	rec := cleanupReceipt{Version: 1, Backup: base, Digest: result.oldDigest}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	path := cleanupReceiptPath(target)
	f, err := openAbsFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return joinRemove(joinClose(err, f), path)
	}
	if err := f.Sync(); err != nil {
		return joinRemove(joinClose(err, f), path)
	}
	if err := f.Close(); err != nil {
		return joinRemove(err, path)
	}
	if err := restrictToCurrentUser(path); err != nil {
		return joinRemove(err, path)
	}
	return nil
}

func restrictToCurrentUser(path string) (err error) {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, token.Close())
	}()
	user, err := token.GetTokenUser()
	if err != nil {
		return err
	}
	sid := user.User.Sid
	dacl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: windows.GENERIC_ALL,
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       windows.NO_INHERITANCE,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_USER,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}}, nil)
	if err != nil {
		return err
	}
	sec, err := windows.NewSecurityDescriptor()
	if err != nil {
		return err
	}
	if err := sec.SetDACL(dacl, true, false); err != nil {
		return err
	}
	if err := sec.SetOwner(sid, false); err != nil {
		return err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(
		abs,
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		sid,
		nil,
		dacl,
		nil,
	)
}
