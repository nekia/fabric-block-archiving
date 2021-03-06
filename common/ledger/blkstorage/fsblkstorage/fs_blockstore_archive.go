/*
Copyright FST & FAST (2018)

Provides public entry points to some previously private methods
*/

package fsblkstorage

import (
	"errors"
	"io"
	"net"
	"os"

	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/hyperledger/fabric/common/ledger/blockarchive"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
)

// sendBlockfileToVault - Moves a blockfile into the vault
func sendBlockfileToVault(cid string, fileNum int) error {

	blockfileDir := filepath.Join(ledgerconfig.GetBlockStorePath(), ChainsDir, cid)
	srcFilePath := deriveBlockfilePath(blockfileDir, fileNum)
	srcFile, err := os.Open(srcFilePath)
	if err != nil {
		logger_ar.Warningf("Already archived : blockfileDir [%s] fileNum [%d]", blockfileDir, fileNum)
		return errors.New("Already archived")
	}
	defer srcFile.Close()

	config := &ssh.ClientConfig{
		User: "root",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Auth: []ssh.AuthMethod{
			ssh.Password("blkstore"),
		},
	}
	config.SetDefaults()
	blockVaultURL := blockarchive.BlockVaultURL
	sshConn, err := ssh.Dial("tcp", blockVaultURL, config)
	if err != nil {
		logger_ar.Warningf("Block store server [%s] is unreachable [%s]", blockVaultURL, err.Error())
		return errors.New("Server unreachable")
	}
	defer sshConn.Close()

	client, err := sftp.NewClient(sshConn)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	blockVaultDir := blockarchive.BlockVaultDir
	dstDirPath := filepath.Join(blockVaultDir, filepath.Dir(srcFilePath))
	dstFilePath := filepath.Join(blockVaultDir, srcFilePath)
	client.MkdirAll(dstDirPath)
	dstFile, err := client.Create(dstFilePath)
	if err != nil {
		panic(err)
	}
	defer dstFile.Close()

	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		panic(err)
	}

	logger_ar.Info("sendBlockfileToVault - sent blockfile to vault: ", fileNum, " written=", written)

	return nil
}

func (mgr *blockfileMgr) notifyArchiver(fileNum int) {
	logger_ar.Info("mgr.notifyArchiver...")
	arChan := mgr.archiverChan
	if arChan != nil {
		logger_ar.Info("mgr.notifyArchiver - sending message...")
		msg := blockarchive.ArchiverMessage{mgr.chainID, fileNum}
		select {
		case arChan <- msg:
		default:
			logger_ar.Warning("mgr.notifyArchiver - message not sent!!!")
		}
	}
}

func (mgr *blockfileMgr) SetArchiverChan(ch chan blockarchive.ArchiverMessage) {
	mgr.archiverChan = ch
}

func (store *fsBlockStore) SetBlockArchived(blockFileNo int, deleteTheFile bool) error {
	return store.archiver.SetBlockfileArchived(blockFileNo, deleteTheFile)
}
