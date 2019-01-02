package cmder

import (
	"bytes"
	"os/exec"
)

func Exec_shell(cmd string) (string, error) {
	//该函数执行一个shell命令，如果成功返回成功信息，err为nil
	//如果执行命令失败，返回shell的stderr提示，且err不为nil（类似exit code 127）
	cmdResult := exec.Command("/bin/bash", "-c", cmd)
	var (
		out    bytes.Buffer
		errOut bytes.Buffer
	)
	// stdout获取标准输出，stderr获取错误输出
	cmdResult.Stdout = &out
	cmdResult.Stderr = &errOut

	if err := cmdResult.Run(); err != nil {
		return errOut.String(), err
	}
	return out.String(), nil
}

//func main() {
//	if len(os.Args)!=0{
//		fmt.Printf("文件路径为：%s\n",os.Args[0])// args 第一个片 是文件路径
//	}
//	s := os.Args[1]
//	res, err := Exec_shell(s)
//	if err != nil {
//		fmt.Printf("命令执行错误，错误信息为：%s\n", res)
//	}else {
//		fmt.Printf("命令执行成功，返回信息为：%s\n", res)
//	}
//}
