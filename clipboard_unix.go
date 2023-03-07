//go:build freebsd || linux || netbsd || openbsd || solaris || dragonfly
// +build freebsd linux netbsd openbsd solaris dragonfly

package clipboard

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
)

// wslで実行されている場合windowsのクリップボードを参照する
func write_win(file string) error {
	cmd := exec.Command("PowerShell.exe", "-Command", "Add-Type", "-AssemblyName",
		fmt.Sprintf("System.Windows.Forms;[Windows.Forms.Clipboard]::SetImage([System.Drawing.Image]::FromFile('%s'));", file))
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(b))
	}
	return nil
}

func read_win() (io.Reader, error) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	f.Close()
	defer os.Remove(f.Name())

	cmd := exec.Command("PowerShell.exe", "-Command", "Add-Type", "-AssemblyName",
		fmt.Sprintf("System.Windows.Forms;$clip=[Windows.Forms.Clipboard]::GetImage();if ($clip -ne $null) { $clip.Save('%s') };", f.Name()))
	b, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err, string(b))
	}

	r := new(bytes.Buffer)
	f, err = os.Open(f.Name())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := io.Copy(r, f); err != nil {
		return nil, err
	}

	return r, nil
}

func isWsl() bool {
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	} else {
		return false
	}
}

func write(file string) error {

	if isWsl() {
		return write_win(file)
	}

	b, err := exec.Command("file", "-b", "--mime-type", file).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(b))
	}

	// b has new line
	cmd := exec.Command("xclip", "-selection", "clipboard", "-t", string(b[:len(b)-1]))
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(in, f); err != nil {
		return err
	}

	if err := in.Close(); err != nil {
		return err
	}

	return cmd.Wait()
}

func read() (io.Reader, error) {

	if isWsl() {
		return read_win()
	}

	cmd := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-o")
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r); err != nil {
		return nil, err
	}

	if err := r.Close(); err != nil {
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return buf, nil
}
