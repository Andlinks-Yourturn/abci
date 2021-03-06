package dummy

import (
	//"strings"

	"github.com/tendermint/abci/types"
	"github.com/tendermint/merkleeyes/iavl"
	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/merkle"
	"github.com/tendermint/go-wire"
	"github.com/bitly/go-simplejson"
	"strconv"
	"net/http"
	"fmt"
	"os/exec"
	"os"
	"io/ioutil"
	"encoding/hex"
)

type DummyApplication struct {
	types.BaseApplication

	state merkle.Tree
}
// Transaction type bytes
const (
	WriteSet byte = 0x01
	WriteRem byte = 0x02
)

//project account 的密码
const (
	PathDoc string = "/Users/b/Documents/"
	pwd string = "1234567890"
	url string ="http://192.168.1.64:46600/"
)
func NewDummyApplication() *DummyApplication {
	state := iavl.NewIAVLTree(0, nil)
	return &DummyApplication{state: state}
}

func (app *DummyApplication) Info() (resInfo types.ResponseInfo) {
	return types.ResponseInfo{Data: cmn.Fmt("{\"size\":%v}", app.state.Size())}
}

// tx is either "key=value" or just arbitrary bytes
func (app *DummyApplication) DeliverTx(tx []byte) types.Result {
	//parts := strings.Split(string(tx), "=")
	//if len(parts) == 2 {
	//	app.state.Set([]byte(parts[0]), []byte(parts[1]))
	//} else {
	//	app.state.Set(tx, tx)
	//}
	//return types.OK
	tree := app.state
	return app.doTx(tree, tx)
}
func (app *DummyApplication) doTx(tree merkle.Tree, tx []byte) types.Result {
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
		//case WriteRem: // Remove
		//	key, n, err := wire.GetByteSlice(tx)
		//	if err != nil {
		//		return types.ErrEncodingError.SetLog(cmn.Fmt("Error reading key: %v", err.Error()))
		//	}
		//	tx = tx[n:]
		//	if len(tx) != 0 {
		//		return types.ErrEncodingError.SetLog(cmn.Fmt("Got bytes left over"))
		//	}
		//	tree.Remove(key)
	case WriteRem: // Compare
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

		//获得IPFS地址
		_, stuValue, stuExists := app.state.Get(value)
		_, pojValue, pojExists := app.state.Get(key)

		//判断两个地址都存存在
		if stuExists && pojExists{
			matched := Compare(string(stuValue),string(pojValue))
			if matched {
				fmt.Println("matched")
				filepath := PathDoc+string(pojValue)
				sendAmount := getIntItem(string(filepath),"amount")
				//若匹配则发出转账申请
				projectname := getProjectName(filepath)
				//toName :="A3AC84A1DB492F2F284BA4CC5DBC933703C7D161"
				rel := sendBasecoinTx(url,projectname,value,sendAmount)
				//sendBasecoinTx(url,"andlinks",toName,sendAmount)
				if rel == "success" {
					return types.NewResultOK([]byte("Matched"), "log")
				}else{
					return types.ErrUnauthorized.SetLog("failed to send ")
				}

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


func (app *DummyApplication) filterTx(tree merkle.Tree, tx []byte) types.Result {
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
		//case WriteRem: // Remove
		//	key, n, err := wire.GetByteSlice(tx)
		//	if err != nil {
		//		return types.ErrEncodingError.SetLog(cmn.Fmt("Error reading key: %v", err.Error()))
		//	}
		//	tx = tx[n:]
		//	if len(tx) != 0 {
		//		return types.ErrEncodingError.SetLog(cmn.Fmt("Got bytes left over"))
		//	}
		//	tree.Remove(key)
	case WriteRem: // Compare
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

		//获得IPFS地址
		_, stuValue, stuExists := app.state.Get(value)
		_, pojValue, pojExists := app.state.Get(key)

		fmt.Println("sdudent: ",stuValue," project: ",pojValue)
		//判断两个地址都存存在
		if stuExists && pojExists{
			matched := Compare(string(stuValue),string(pojValue))
			if matched {

				return types.NewResultOK([]byte("Matched"),"success")

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




//send basecoin transaction to server
func sendBasecoinTx(url string,from string,to []byte,amount int) string{
	//http://192.168.1.64:46600/sendTx?userFrom=andlinks&password=1234567890&money=1000mycoin&userToAddress=A3AC84A1DB492F2F284BA4CC5DBC933703C7D161
	result := hex.EncodeToString(to)
	fmt.Println("send to ",result)
	request := url+"sendTx?userFrom="+from+"&password="+pwd+"&money="+strconv.Itoa(amount)+"mycoin&userToAddress="+result
	fmt.Println("url, ",request)
	res, err := http.Get(request)
	if err != nil{
		panic(err)
	}
	body, err := ioutil.ReadAll(res.Body)

	if string(body) == "true" {
		fmt.Println("ssuccess")
		return "success"
	}else{
		fmt.Println("error")
		return "failure"
	}



}
func (app *DummyApplication) CheckTx(tx []byte) types.Result {
	//return types.OK
	tree := app.state
	return app.filterTx(tree, tx)
}

func (app *DummyApplication) Commit() types.Result {
	hash := app.state.Hash()
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

func getProjectName(path string) string{

	dat, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	//fmt.Print(string(dat))
	js, err :=simplejson.NewJson([]byte(dat))
	name := js.Get("name").MustString()
	fmt.Println("---name---",name)
	return name
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
	fmt.Println("rank C: ",rankC)

	rankS := getIntItem(target,"rank")
	fmt.Println("rank S: ",rankS)

	//专业一致
	majorC := getStringItem(criteria,"major")
	fmt.Println("major C : ",majorC)

	majorS := getStringItem(target,"major")
	fmt.Println("major S: ",majorS)

	if rankC>rankS && majorC == majorS {
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
func (app *DummyApplication) Query(reqQuery types.RequestQuery) (resQuery types.ResponseQuery) {
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
		index, value, exists := app.state.Get(reqQuery.Data)
		resQuery.Index = int64(index)
		resQuery.Value = value
		if exists {
			resQuery.Log = "exists"
		} else {
			resQuery.Log = "does not exist"
		}
		return
	}
}
