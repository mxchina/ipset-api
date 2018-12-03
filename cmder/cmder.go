package cmder

import (
	"bytes"
	"os/exec"
)

func Exec_shell(s string) (string, error) {
	//函数返回一个*Cmd，用于使用给出的参数执行name指定的程序
	cmd := exec.Command("/bin/bash", "-c", s)

	//读取io.Writer类型的cmd.Stdout，再通过bytes.Buffer(缓冲byte类型的缓冲器)将byte类型转化为string类型(out.String():这是bytes类型提供的接口)
	var out bytes.Buffer
	cmd.Stdout = &out

	//Run执行c包含的命令，并阻塞直到完成。  这里stdout被取出，cmd.Wait()无法正确获取stdin,stdout,stderr，则阻塞在那了
	err := cmd.Run()
	return out.String(), err
}

//func main() {
//	if len(os.Args)!=0{
//		fmt.Println(os.Args[0])// args 第一个片 是文件路径
//	}
//	s := os.Args[1]
//	res, err := Exec_shell(s)
//	if err != nil {
//		fmt.Println(err)
//	}
//	fmt.Println("结果：")
//	fmt.Printf("%q",res)
//}
