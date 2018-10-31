/*
 * go4api - a api testing tool written in Go
 * Created by: Ping Zhu 2018
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 *
 */

package api

import (
    "fmt"
    "os"
    "bytes"
    "mime/multipart"
    "io"    
    "net/url"  
    "strings"
    "encoding/json"

    "go4api/cmd"
    "go4api/lib/testcase" 

    gjson "github.com/tidwall/gjson"
)

func GetPayloadInfo (tcData testcase.TestCaseDataInfo) (string, string, *strings.Reader, *bytes.Buffer, string) {
    apiMethod := tcData.TestCase.ReqMethod()
    // request payload(body)
    reqPayload := tcData.TestCase.ReqPayload()
    //
    var bodyText *strings.Reader // init body
    bodyMultipart := &bytes.Buffer{}
    boundary := ""
    //
    apiMethodSelector := apiMethod
    // Note, has 3 conditions: text (json), form, or multipart file upload
    for key, _ := range reqPayload {
        switch key {
            case "form-data": {
                //multipart/form-data
                apiMethodSelector = "POSTMultipart"
                multipartFilePath := cmd.Opt.Testresource

                if string(multipartFilePath[len(multipartFilePath) - 1]) != "/" {
                    multipartFilePath = multipartFilePath + "/"
                }

                bodyMultipart, boundary, _ = PrepMultipart(reqPayload, multipartFilePath)
            }
            case "text": {
                //application/json
                bodyText = PrepPostPayload(reqPayload)
            }
            case "form": {
                //application/x-www-form-urlencoded
                bodyText = PrepPostFormPayload(reqPayload)
            }
            default: {
                bodyText = strings.NewReader("")
            }
        }
    }

    return apiMethodSelector, apiMethod, bodyText, bodyMultipart, boundary
}

func fileOpen (path string, fileName string) *os.File {
    fp, err := os.Open(path + fileName) 
    if err != nil {
        panic(err)
    }

    return fp
}

func PrepMultipart (reqPayload map[string]interface {}, path string) (*bytes.Buffer, string, error) {
    body := &bytes.Buffer{} // init body
    writer := multipart.NewWriter(body) // multipart
    
    // prepare the reader instances to encode
    reqPayloadJsonBytes, _ := json.Marshal(reqPayload)
    reqPayloadJson := string(reqPayloadJsonBytes)

    var params = make(map[string]io.Reader)
    var i int64
    total := gjson.Get(reqPayloadJson, "form-data.#")
    for i = 0; i < total.Int(); i++ {
        name := gjson.Get(reqPayloadJson, "form-data." + fmt.Sprint(i) + ".name")
        fieldType := gjson.Get(reqPayloadJson, "form-data." + fmt.Sprint(i) + ".type")
        value := gjson.Get(reqPayloadJson, "form-data." + fmt.Sprint(i) + ".value")

        if strings.ToLower(fieldType.String()) == "file" {
            fp := fileOpen(path, value.String())
            defer fp.Close()

            params[name.String()] = fp
        } else {
            params[name.String()] = strings.NewReader(value.String())
        }
    }
    //
    var err error
    for key, r := range params {
        var fw io.Writer

        if x, ok := r.(io.Closer); ok {
            defer x.Close()
        }
        // Add an file
        if x, ok := r.(*os.File); ok {
            if fw, err = writer.CreateFormFile(key, x.Name()); err != nil {
                return nil, "", err
            }
        } else {
            // Add other fields
            if fw, err = writer.CreateFormField(key); err != nil {
                return nil, "", err
            }
        }
        //
        if _, err = io.Copy(fw, r); err != nil {
            panic(err)
            return nil, "", err
        }
    }
    //
    err = writer.Close()
    if err != nil {
        return nil, "", err
    }
    // do not forget this
    boundary := writer.FormDataContentType()
    // fmt.Println("boundary", boundary)
    // ==> i.e. multipart/form-data; boundary=37b1e9deba0159aaf429d7183a9de344c532e50299532f7b4f7bdbbca435

    return body, boundary, nil

}


func PrepPostPayload (reqPayload map[string]interface{}) *strings.Reader {
    var body *strings.Reader

    for key, value := range reqPayload {
        if key == "text" {
            repJson, _ := json.Marshal(value)
            body = strings.NewReader(string(repJson))
            break
        }
    }

    return body
}

func PrepPostFormPayload (reqPayload map[string]interface{}) *strings.Reader {
    var body *strings.Reader

    data := url.Values{}
    for key, value := range reqPayload {
        // value (type interface {}) as type string in argument to data.Set: need type assertion
        data.Set(key, fmt.Sprint(value))
    }
    body = strings.NewReader(data.Encode())

    return body
}
