package diambra

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/diambra/cli/pkg/diambra/client"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"golang.org/x/term"
)

const LoginBanner = `------------------------------------------------

	DIAMBRA Arena is completely free to use,
 you only need to register (in a few clicks) at
		  https://diambra.ai/register/

------------------------------------------------`

func Login(dc *client.Client, credPath string) error {
	bp := filepath.Dir(credPath)
	if err := os.MkdirAll(bp, 0755); err != nil {
		return fmt.Errorf("can't create %s: %w", bp, err)
	}

	var username string
	fmt.Println(LoginBanner)
	fmt.Print("Username (or Email): ")
	if _, err := fmt.Scanln(&username); err != nil {
		return fmt.Errorf("couldn't read username: %w", err)
	}
	fmt.Print("Password: ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("couldn't read password: %w", err)
	}

	token, err := dc.Token(username, string(password))
	if err != nil {
		return fmt.Errorf("couldn't get token. Invalid password?: %w", err)
	}
	fmt.Println("token", token, "credPath", credPath)
	fh, err := os.OpenFile(credPath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("can't create credentials file %s: %w", credPath, err)
	}
	_, err = fmt.Fprint(fh, token)
	if err != nil {
		return fmt.Errorf("couldn't write credentials file %s: %w", credPath, err)
	}
	fh.Close()
	return nil
}

func EnsureCredentials(logger log.Logger, credPath string) error {
	exists, isDir := pathExistsAndIsDir(credPath)
	if exists && isDir {
		return fmt.Errorf("path.credentials %s is a directory. Is --path.credentials set correctly?", credPath)
	}

	dc, err := client.NewClient(logger, credPath)
	if err != nil {
		return fmt.Errorf("couldn't create client: %w", err)
	}

	if exists {
		var err error
		if err != nil {
			return fmt.Errorf("couldn't create client: %w", err)
		}
		user, err := dc.User()
		if err == nil {
			level.Info(logger).Log("msg", "logged in", "user", user.Username)
			return nil
		}

		if _, ok := err.(client.ErrForbidden); ok {
			if err := os.Remove(credPath); err != nil {
				return fmt.Errorf("couldn't remove credentials file %s: %w", credPath, err)
			}
			level.Warn(logger).Log("msg", "Invalid credentials, please login again")
			return Login(dc, credPath)
		}
		return err
	}

	return Login(dc, credPath)
}
