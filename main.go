// golang_pkg_auto_fix project main.go
package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"netutil"
	"os"
	"strings"
	"time"
	"toolfunc"
	"unicode"

	"github.com/CodyGuo/win"
)

var goroot string
var logf *os.File

var (
	winExecError = map[uint32]string{
		0:  "The system is out of memory or resources.",
		2:  "The .exe file is invalid.",
		3:  "The specified file was not found.",
		11: "The specified path was not found.",
	}
)

func execRun(cmd string) error {
	lpCmdLine := win.StringToBytePtr(cmd)
	ret := win.WinExec(lpCmdLine, win.SW_HIDE)
	if ret <= 31 {
		return errors.New(winExecError[ret])
	}

	return nil
}

func getsevenzpath() string {
	if toolfunc.IsFileExists("C:\\Program Files\\7-Zip\\7z.exe") {
		return "C:\\Program Files\\7-Zip\\7z.exe"
	} else if toolfunc.IsFileExists("C:\\Program Files (x86)\\7-Zip\\7z.exe") {
		return "C:\\Program Files (x86)\\7-Zip\\7z.exe"
	} else {
		curdir := toolfunc.CurDir()
		tooldir7z := curdir[0:strings.Index(curdir, "\\")+1] + "work\\" + "tool\\7-Zip\\7z.exe"
		if toolfunc.IsFileExists(tooldir7z) {
			return tooldir7z
		}
	}
	return ""
}

func getdirgofiletime(dir string) time.Time {
	if !(dir[len(dir)-1] == '/' || dir[len(dir)-1] == '\\') {
		dir += "/"
	}
	dirinfo, err := ioutil.ReadDir(dir)
	if err == nil {
		for _, di := range dirinfo {
			if di.IsDir() {
				t1 := getdirgofiletime(dir + di.Name())
				if t1.Second() != 0 {
					return t1
				}
			} else {
				if strings.HasSuffix(di.Name(), ".go") {
					fstat, _ := os.Stat(dir + di.Name())
					return fstat.ModTime()
				}
			}
		}
	}
	return toolfunc.TimeFromInt64(0)
}

func checkpkgneedupdate(pkgpath, pkg string) bool {
	//https://api.github.com/repos/tinode/jsonco
	//"updated_at": "2020-08-08T01:23:56Z",
	pkgsub := pkg[strings.Index(pkg, "/")+1:]
	ctt, _, _, httpcode, _ := netutil.UrlGet("https://api.github.com/repos/"+pkgsub, nil, false, nil, nil, 30*time.Second, 30*time.Second, nil)
	if httpcode == 200 {
		//toolfunc.WriteFile("aaa.txt", ctt)
		cttstr := string(ctt)
		l1 := strings.Index(cttstr, "\"updated_at\":\"")
		cttstr = cttstr[l1+len("\"updated_at\":\""):]
		cttstr = cttstr[:strings.Index(cttstr, "Z")]
		t1 := toolfunc.IsoTimeToTime(cttstr)
		t2 := getdirgofiletime(pkgpath)
		if t1.Second() > 0 && t2.Second() > 0 && t1.Second()-t2.Second() > 0 {
			return true
		}
	}
	return false
}

func WalkDir(dir string) {
	if !(dir[len(dir)-1] == '/' || dir[len(dir)-1] == '\\') {
		dir += "/"
	}
	dirinfo, err := ioutil.ReadDir(dir)
	if err == nil {
		for _, di := range dirinfo {
			if di.IsDir() {
				WalkDir(dir + di.Name())
			} else {
				if di.Name() == "go.mod" {
					ctt3, err2 := ioutil.ReadFile(dir + di.Name())
					ctt := string(ctt3)
					if err2 == nil {
						l1 := strings.Index(ctt, "require (")
						l2 := strings.LastIndex(ctt, ")")
						fmt.Println(l1, l2, string(ctt))
						if l1 == -1 || l2 == -1 {
							continue
						}
						ctt2 := ctt[l1+len("require (") : l2]
						lns := strings.Split(ctt2, "\n")
						for _, ln := range lns {
							lnstr := strings.Trim(ln, " \r\n\t")
							i3 := strings.Index(lnstr, " ")
							if i3 != -1 && (unicode.IsUpper(rune(lnstr[0])) || unicode.IsLower(rune(lnstr[0]))) {
								pkgpath := lnstr[0:i3]
								if strings.HasPrefix(pkgpath, "github.com/") {
									ppath := goroot + pkgpath + "/"
									ppdirinfo, err4 := ioutil.ReadDir(ppath)
									var bupdate bool
									if !(!toolfunc.IsDirExists(ppath) || err4 == nil && len(ppdirinfo) == 0) {
										bupdate = checkpkgneedupdate(ppath, pkgpath)
									}
									if !toolfunc.IsDirExists(ppath) || err4 == nil && len(ppdirinfo) == 0 || bupdate {
										if bupdate {
											logf.Write([]byte("update:" + ppath + "\n"))
											toolfunc.RemoveDirAll(ppath)
										} else {
											logf.Write([]byte("download:" + ppath + "\n"))
										}
										logf.Sync()
										toolfunc.MakePathDirExists(ppath)
										pkgname := pkgpath[strings.LastIndex(pkgpath, "/")+1:]
										//https://codeload.github.com/felixge/httpsnoop/zip/master
										fmt.Println("download:" + "https://codeload." + pkgpath + "/zip/master")
										_, _, httpcode, redi := netutil.UrlGetToFile("https://codeload."+pkgpath+"/zip/master", nil, false, nil, nil, goroot+pkgpath+"/"+pkgname+".zip", 30*time.Second, 30*time.Second)
										fmt.Println("redi:", redi, "httpcode:", httpcode)
										if httpcode == 200 {
											sevenzpath := getsevenzpath()
											zerr := execRun("\"" + sevenzpath + "\" x -o\"" + goroot + pkgpath + "\" \"" + goroot + pkgpath + "/" + pkgname + ".zip\"")
											if zerr == nil {
												time.Sleep(1 * time.Second)
												toolfunc.MoveDir(goroot+pkgpath+"/"+pkgname+"-master", goroot+pkgpath)
												if !toolfunc.IsFileExists(goroot + pkgpath + "/" + pkgname + ".zip") {
													fmt.Println("file didn't exists:", goroot+pkgpath+"/"+pkgname+".zip")
												} else {
													// finfo, finfe := os.Stat(goroot + pkgpath)
													// if finfe == nil {
													// 	dirinfo = append(dirinfo, finfo)
													// }
												}
												os.Remove(goroot + pkgpath + "/" + pkgname + ".zip")
											} else {
												fmt.Println("7z error:", zerr)
											}
										}
									}
								} else {
									ppath := goroot + pkgpath + "/"
									ppdirinfo, err4 := ioutil.ReadDir(ppath)
									if !toolfunc.IsDirExists(ppath) || err4 == nil && len(ppdirinfo) == 0 {
										logf.Write([]byte("lose:" + ppath + "\n"))
										logf.Sync()
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

func main() {
	goroot = toolfunc.CurParentDir()
	if !(goroot[len(goroot)-1] == '/' || goroot[len(goroot)-1] == '\\') {
		goroot += "/"
	}
	if !toolfunc.IsDirExists(goroot + "github.com") {
		fmt.Println("in program parent dir of goroot the github.com directory didn't exists.")
		return
	}
	if len(os.Args) == 1 {
		for i := 20; i >= 0; i-- {
			s := fmt.Sprintf("Go root all package updating in %d seconds ....", i)
			fmt.Printf("\r%s", s)
			time.Sleep(time.Second)
		}
		logf, _ := os.OpenFile("golang_pgk_auto_fix.log", os.O_WRONLY|os.O_CREATE, 0666)
		logf.Seek(0, os.SEEK_END)
		WalkDir(goroot)
		logf.Close()
	} else if toolfunc.IsDirExists(os.Args[1]) {
		logf, _ := os.OpenFile("golang_pgk_auto_fix.log", os.O_WRONLY|os.O_CREATE, 0666)
		logf.Seek(0, os.SEEK_END)
		WalkDir(os.Args[1])
		logf.Close()
	} else {
		fmt.Println("parameter: [package or goroot path]")
	}
}
