package kvstore

import (
	//"strings"

	"github.com/tendermint/abci/types"
	"github.com/tendermint/merkleeyes/iavl"
	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/merkle"
	"github.com/tendermint/go-wire"
	"net/http"
	"github.com/bitly/go-simplejson"
	"fmt"
	"os/exec"
	"os"
	"io/ioutil"
	"time"
	"strconv"
)

type StorageApplication struct {
	types.BaseApplication

	state merkle.Tree//存储 address-ipfs
	projects map[string]merkle.Tree // 存储address-project 记录每个项目的申请状况
	sendlist []transaction
}

type transaction struct {
	Input    string
	Output     string
	Amount   int
}

// Transaction type bytes
const (
	WriteSet byte = 0x01
	WriteRem byte = 0x02
)

const (
	PathDoc string = "/Users/b/Documents/"
	url string ="http://localhost:46600"
)
func NewStorageApplication() *StorageApplication {
	state := iavl.NewIAVLTree(0, nil)
	projects := make(map[string]merkle.Tree)
	sendlist := []transaction{}
	return &StorageApplication{state: state,projects:projects,sendlist:sendlist}
}

func (app *StorageApplication) Info() (resInfo types.ResponseInfo) {
	return types.ResponseInfo{Data: cmn.Fmt("{\"size\":%v}", app.state.Size())}
}

// tx is either "0x01|len(len(key))|len(key)|key|len(len(value))|len(value)|value"
//or "0x02|len(len(key))|len(key)|key|len(len(value))|len(value)|value"
func (app *StorageApplication) DeliverTx(tx []byte) types.Result {
	//parts := strings.Split(string(tx), "=")
	//if len(parts) == 2 {
	//	app.state.Set([]byte(parts[0]), []byte(parts[1]))
	//} else {
	//	app.state.Set(tx, tx)
	//}
	//return types.OK
	tree := app.state

	history := app.projects

	sendlist := app.sendlist
	return app.doTx(sendlist,tree,history, tx)
}
func (app *StorageApplication) doTx(sendlist []transaction,tree merkle.Tree,projects map[string]merkle.Tree, tx []byte) types.Result {
	if len(tx) == 0 {
		return types.ErrEncodingError.SetLog("Tx length cannot be zero")
	}
	typeByte := tx[0]
	tx = tx[1:]
	switch typeByte {
	case WriteSet: // Set
		key, n, err := wire.GetByteSlice(tx)
		if err != nil {
			return types.ErrEncodingError.SetLog(cmn.Fmt("Error reading key: %v", err.Error()))
		}
		tx = tx[n:]
		value, n, err := wire.GetByteSlice(tx)
		if err != nil {
			return types.ErrEncodingError.SetLog(cmn.Fmt("Error reading value: %v", err.Error()))
		}
		tx = tx[n:]
		if len(tx) != 0 {
			return types.ErrEncodingError.SetLog(cmn.Fmt("Got bytes left over"))
		}

		tree.Set(key, value)

	case WriteRem: // Compare 比较申请人是否符合要求
		key, n, err := wire.GetByteSlice(tx)
		if err != nil {
			return types.ErrEncodingError.SetLog(cmn.Fmt("Error reading key: %v", err.Error()))
		}
		tx = tx[n:]
		value, n, err := wire.GetByteSlice(tx)
		if err != nil {
			return types.ErrEncodingError.SetLog(cmn.Fmt("Error reading value: %v", err.Error()))
		}
		tx = tx[n:]
		if len(tx) != 0 {
			return types.ErrEncodingError.SetLog(cmn.Fmt("Got bytes left over"))
		}

		//判断是否重复申请
		// 查找键值是否存在
		if v, ok := projects[string(key)]; ok {
			//存在申请历史，需要比对是否重复申请
			fmt.Println("browsing history")
			project := v
			_, creation, exists := project.Get(value)
			if exists {
				fmt.Println("Applied before")
				return types.ErrEncodingError.SetLog(cmn.Fmt("Applied before @",creation))
			} else{
				//不存在申请历史，插入信息
				fmt.Println("Applied now")
				now := time.Now()

				project.Set(value,[]byte(now.String()))

			}
		} else {
			//不存在申请历史，插入信息
			fmt.Println("Key Not Found")
			newTree := iavl.NewIAVLTree(0, nil)
			now := time.Now()
			newTree.Set(value,[]byte(now.String()))
			projects[string(key)] = newTree


		}

		//获得IPFS地址
		_, stuValue, stuExists := app.state.Get(value)
		_, pojValue, pojExists := app.state.Get(key)

		//判断两个地址都存存在
		if stuExists && pojExists{
			matched := Compare(string(stuValue),string(pojValue))
			if matched {
				_,apptime,_ :=app.projects[string(key)].Get(value)
				fmt.Println("matched",string(apptime))

				//创建返回对象

				filepath := PathDoc+string(pojValue)
				sendAmount := getIntItem(string(filepath),"amount")
				tx := transaction {
					Input: string(key),
					Output:string(value),
					Amount: sendAmount,
				}

				sendlist = append(sendlist, tx)

				return types.NewResultOK([]byte("Matched"),"log")
			}else{
				fmt.Println("not matched")
				return types.OK
			}
		} else {
			return types.ErrUnknownRequest.SetLog(cmn.Fmt("Unexpected Account %X", key))

		}


	default:
		return types.ErrUnknownRequest.SetLog(cmn.Fmt("Unexpected Tx type byte %X", typeByte))
	}
	return types.OK
}

//判断是否有重复申请
func (app *StorageApplication) filterTx(tree merkle.Tree, projects map[string]merkle.Tree,tx []byte) types.Result {
	if len(tx) == 0 {
		return types.ErrEncodingError.SetLog("Tx length cannot be zero")
	}
	typeByte := tx[0]
	tx = tx[1:]
	switch typeByte {
	case WriteSet: // Set
		key, n, err := wire.GetByteSlice(tx)
		if err != nil {
			return types.ErrEncodingError.SetLog(cmn.Fmt("Error reading key: %v", err.Error()))
		}
		tx = tx[n:]
		value, n, err := wire.GetByteSlice(tx)
		if err != nil {
			return types.ErrEncodingError.SetLog(cmn.Fmt("Error reading value: %v", err.Error()))
		}
		tx = tx[n:]
		if len(tx) != 0 {
			return types.ErrEncodingError.SetLog(cmn.Fmt("Got bytes left over"))
		}

		tree.Set(key, value)

	case WriteRem: // Compare 比较申请人是否符合要求
		key, n, err := wire.GetByteSlice(tx)
		if err != nil {
			return types.ErrEncodingError.SetLog(cmn.Fmt("Error reading key: %v", err.Error()))
		}
		tx = tx[n:]
		value, n, err := wire.GetByteSlice(tx)
		if err != nil {
			return types.ErrEncodingError.SetLog(cmn.Fmt("Error reading value: %v", err.Error()))
		}
		tx = tx[n:]
		if len(tx) != 0 {
			return types.ErrEncodingError.SetLog(cmn.Fmt("Got bytes left over"))
		}

		//判断是否重复申请
		// 查找键值是否存在
		if v, ok := projects[string(key)]; ok {
			//存在申请历史，需要比对是否重复申请
			fmt.Println("browsing history")
			project := v
			_, creation, exists := project.Get(value)
			if exists {
				fmt.Println("Applied before")
				return types.ErrEncodingError.SetLog(cmn.Fmt("Applied before @",creation))
			}
		}

		//获得IPFS地址
		_, _, stuExists := app.state.Get(value)
		_, _, pojExists := app.state.Get(key)

		//判断两个地址都存存在
		if stuExists && pojExists{
			return types.NewResultOK([]byte("Ready to compare documents "),"log")
		} else {
			return types.ErrUnknownRequest.SetLog(cmn.Fmt("Unexpected Account %X", key,"and %X", value))

		}


	default:
		return types.ErrUnknownRequest.SetLog(cmn.Fmt("Unexpected Tx type byte %X", typeByte))
	}
	return types.OK
}

func (app *StorageApplication) CheckTx(tx []byte) types.Result {
	//return types.OK
	tree := app.state
	history := app.projects

	return app.filterTx(tree,history, tx)
}

func (app *StorageApplication) Commit() types.Result {

	//将 sendlist中记录的交易执行
	for _, elem := range app.sendlist {

		result := sendBasecoinTx(url,elem.Input,elem.Output,elem.Amount)
		fmt.Println("send result from %s,to %s, with %d, the result is %s",elem.Input,elem.Output,elem.Amount,result)
	}
	app.sendlist = nil
	hash := app.state.Hash()
	fmt.Println("commit transaction")
	return types.NewResultOK(hash, "")
}
func ipfsDownload(add string,path string) {

	var (
		cmdOut []byte
		err    error
	)

	cmd := "ipfs"

	args := []string{"get",add,"-o",path}

	if cmdOut, err = exec.Command(cmd, args...).Output(); err != nil {
		fmt.Fprintln(os.Stderr, "There was an error running git rev-parse command: ", err)
		os.Exit(1)
	}
	result := string(cmdOut)

	fmt.Println(result)
	//fmt.Println("get result",sequenceInt)
}


func getIntItem(path string,item string) int{

	dat, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	//fmt.Print(string(dat))
	js, err :=simplejson.NewJson([]byte(dat))
	majorStr := js.Get("criteria").Get(item).MustInt()
	fmt.Println("---",majorStr)
	return majorStr
}

func getStringItem(path string,item string) string{

	dat, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	//fmt.Print(string(dat))
	js, err :=simplejson.NewJson([]byte(dat))
	majorStr := js.Get("criteria").Get(item).MustString()
	fmt.Println("---",majorStr)
	return majorStr
}

//send basecoin transaction to server
func sendBasecoinTx(url string,from string,to string,amount int) string{

	//http://192.168.1.64:46600/sendTx?userFrom=&password=&money=&userToAddress"
	request := url+"sendTx?userFrom="+from+"&money="+strconv.Itoa(amount)+"&userToAddress"+to
	fmt.Println("url, ",request)
	res, err := http.Get(request)
	if err != nil{
		panic(err)
	}
	body, err := ioutil.ReadAll(res.Body)

	if string(body) == "true" {
		fmt.Println("success")
		return "success"
	}else{
		fmt.Println("error")
		return "failure"
	}



}

/*
比较提交的文件是否与要求匹配
input: criteria 条件json ；target：申请人条件文件
output: 结果 bool
*/
func compareFiles(criteria string, target string) bool{

	////年龄小于限定值
	//ageC := getIntItem(criteria,"age")
	//fmt.Println("age: ",ageC)
	//
	//ageS := getIntItem(target,"age")
	//fmt.Println("age: ",ageS)

	// 排名在要求之前
	rankC := getIntItem(criteria,"rank")
	fmt.Println("rank: ",rankC)

	rankS := getIntItem(target,"rank")
	fmt.Println("rank: ",rankS)

	//专业一致
	majorC := getStringItem(criteria,"major")
	fmt.Println("major: ",majorC)

	majorS := getStringItem(target,"major")
	fmt.Println("major: ",majorS)

	if rankC>rankS && majorC ==majorS {
		return true
	}else{
		return false
	}


}

func Compare(studentAdd string,projectAdd string) bool{

	ipfsDownload(studentAdd,PathDoc)
	ipfsDownload(projectAdd,PathDoc)

	filepath2 := PathDoc+studentAdd
	filepath := PathDoc+projectAdd

	result := compareFiles(filepath,filepath2)
	fmt.Println("get result", result)
	//return types.NewResultOK([]byte("OK"), "")
	return result
}
func (app *StorageApplication) Query(reqQuery types.RequestQuery) (resQuery types.ResponseQuery) {
	if reqQuery.Prove {
		value, proof, exists := app.state.Proof(reqQuery.Data)
		resQuery.Index = -1 // TODO make Proof return index
		resQuery.Key = reqQuery.Data
		resQuery.Value = value
		resQuery.Proof = proof
		if exists {
			resQuery.Log = "exists"
		} else {
			resQuery.Log = "does not exist"
		}
		return
	} else {
		switch reqQuery.Path {
		case "tree":
			index, value, exists := app.state.Get(reqQuery.Data)
			resQuery.Index = int64(index)
			resQuery.Value = value
			if exists {
				resQuery.Log = "exists"
			} else {
				resQuery.Log = "does not exist"
			}

		case "projects":
			tree := app.projects[string(reqQuery.Data)]

			//  traversing the whole node works... in order
			//viewed := []string{}
			viewed := ""

			if tree != nil {
				length := tree.Height()
				fmt.Println("tree size = ",length)
				tree.Iterate(func(key []byte, value []byte) bool {
					viewed = viewed+" , "+string(key)
					return false
				})
				resQuery.Value = []byte(viewed)
			}else {
				resQuery.Value = []byte(strconv.Itoa(0))
			}


		default:
			return types.ResponseQuery{Log: cmn.Fmt("Invalid query path. Expected hash or tx, got %v", reqQuery.Path)}
		}
		return
	}

}
