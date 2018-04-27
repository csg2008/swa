package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SafeFileName replace all illegal chars to a underline char
func SafeFileName(fileName string) string {
	return strings.Map(func(r rune) rune {
		if strings.IndexRune(`/\:*?"><|`, r) != -1 {
			return '_'
		}
		return r
	}, fileName)
}

// GetDirFiles 获取指定文件夹文件列表
func GetDirFiles(dirPath string, stripExt bool) []string {
	var ret []string

	if files, err := ioutil.ReadDir(dirPath); nil == err && nil != files {
		ret = make([]string, 0, len(files))
		for _, v := range files {
			if !v.IsDir() {
				if stripExt {
					var tmp = v.Name()
					var idx = strings.LastIndexByte(v.Name(), '.')
					if idx > 0 {
						ret = append(ret, strings.Trim(tmp[0:idx], " "))
					} else {
						ret = append(ret, v.Name())
					}
				} else {
					ret = append(ret, v.Name())
				}
			}
		}
	}

	return ret
}

// WalkRelFiles 遍历文件，可指定后缀，返回相对路径
func WalkRelFiles(target string, suffixes ...string) (files []string) {
	if !filepath.IsAbs(target) {
		target, _ = filepath.Abs(target)
	}
	err := filepath.Walk(target, func(retpath string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if len(suffixes) == 0 {
			files = append(files, RelPath(retpath))
			return nil
		}
		_retpath := RelPath(retpath)
		for _, suffix := range suffixes {
			if strings.HasSuffix(_retpath, suffix) {
				files = append(files, _retpath)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("function.WalkRelFiles: %v\n", err)
		return
	}

	return
}

// WalkRelDirs 遍历目录，可指定后缀，返回相对路径
func WalkRelDirs(target string, suffixes ...string) (dirs []string) {
	if !filepath.IsAbs(target) {
		target, _ = filepath.Abs(target)
	}
	err := filepath.Walk(target, func(retpath string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !f.IsDir() {
			return nil
		}
		if len(suffixes) == 0 {
			dirs = append(dirs, RelPath(retpath))
			return nil
		}
		_retpath := RelPath(retpath)
		for _, suffix := range suffixes {
			if strings.HasSuffix(_retpath, suffix) {
				dirs = append(dirs, _retpath)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("utils.WalkRelDirs: %v\n", err)
		return
	}

	return
}

// IsFile returns true if given path is a file,
// or returns false when it's a directory or does not exist.
func IsFile(filePath string) bool {
	f, e := os.Stat(filePath)
	if e != nil {
		return false
	}
	return !f.IsDir()
}

// IsDir returns true if given path is a directory,
func IsDir(filePath string) bool {
	f, e := os.Stat(filePath)
	if e != nil {
		return false
	}
	return f.IsDir()
}

// IsExist checks whether a file or directory exists.
// It returns false when the file or directory does not exist.
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// Dirname 返回指定路径的文件夹名
func Dirname(target string) string {
	idx := strings.LastIndex(strings.TrimRight(target, "/"), "/")

	if -1 != idx {
		return target[:idx]
	}

	return ""
}

// RelPath 转相对路径
func RelPath(target string) string {
	basePath, _ := filepath.Abs("./")
	rel, _ := filepath.Rel(basePath, target)
	return strings.Replace(rel, "\\", "/", -1)
}

// AbsPath 转换相对路径为绝对路径
func AbsPath(target string) string {
	basePath, _ := filepath.Abs(target)
	return strings.Replace(basePath, "\\", "/", -1)
}

// FileExt 返回文件扩展名
func FileExt(path string) string {
	return filepath.Ext(path)
}

// GetAppPath 返回应用程序当前路径
func GetAppPath() string {
	var curPath, _ = exec.LookPath(os.Args[0])

	return strings.Replace(filepath.Dir(AbsPath(curPath)), "\\", "/", -1)
}

// FileGetContents Get bytes to file.
// if non-exist, create this file.
func FileGetContents(filename string) (data []byte, e error) {
	f, e := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if e != nil {
		return
	}
	defer func() {
		_ = f.Close()
	}()

	stat, e := f.Stat()
	if e != nil {
		return
	}
	data = make([]byte, stat.Size())
	result, e := f.Read(data)
	if e != nil || int64(result) != stat.Size() {
		return nil, e
	}
	return
}

// FilePutContents Put bytes to file.
// if non-exist, create this file.
func FilePutContents(filename string, content []byte, append bool) error {
	var flag int
	if append {
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	} else {
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	fp, err := os.OpenFile(filename, flag, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() {
		_ = fp.Close()
	}()

	_, err = fp.Write(content)
	return err
}
