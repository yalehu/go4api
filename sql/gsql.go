/*
 * go4api - a api testing tool written in Go
 * Created by: Ping Zhu 2018
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 *
 */

package gsql

import (
    "fmt"
    "os"
    // "strconv"
    "strings"
    "database/sql"
    // "encoding/json"

    "go4api/cmd"

    _ "github.com/go-sql-driver/mysql"
)

// var db = &sql.DB{}
var SqlCons map[string]*sql.DB

type SqlExec struct {
    TargetDb string
    Stmt string
    RowsCount int
    RowsHeaders []string
    RowsData []map[string]interface{}
}

func InitConnection () {
    SqlCons = make(map[string]*sql.DB)

    dbs := cmd.GetDbConfig()

    for k, v := range dbs {
        ip := v.Ip
        port := v.Port
        user := v.UserName
    
        pw := ""
        pwV := v.Password
        pwV = strings.Replace(pwV, "${", "", -1)
        pwV = strings.Replace(pwV, "}", "", -1)
        if len(pwV) > 0 {
            pw = os.Getenv(pwV)
        }
        
        defaultSchema := os.Getenv("go4_dev_db_defaultSchema")

        conInfo := user + ":" + pw + "@tcp(" + ip + ":" + fmt.Sprint(port) + ")/" + defaultSchema
        db, _ := sql.Open("mysql", conInfo)
        db.SetMaxOpenConns(2000)
        db.SetMaxIdleConns(1000)

        err := db.Ping()
        if err != nil {
            panic(err)
        }

        key := strings.ToLower(k)
        SqlCons[key] = db
    }
} 

func Run (stmt string) (int, []string, []map[string]interface{}, string) {
    // update, delete, select, insert
    s := strings.TrimSpace(stmt)
    s = strings.ToUpper(s)
    s = string([]rune(stmt)[:6])

    var err error
    sqlExecStatus := ""

    fmt.Println("sqlcmd: ", stmt)

    tDb := "master"
    sqlExec := &SqlExec{tDb, stmt, 0, []string{}, []map[string]interface{}{}}

    switch strings.ToUpper(s) {
        case "UPDATE":
            err = sqlExec.Update()
        case "DELETE":
            err = sqlExec.Delete()
        case "SELECT":
            err = sqlExec.QueryWithoutParams()
        case "INSERT":
            err = sqlExec.Insert() 
    }

    if err == nil {
        sqlExecStatus = "cmdSuccess"
    } else {
        sqlExecStatus = "cmdFailed"
    }

    return sqlExec.RowsCount, sqlExec.RowsHeaders, sqlExec.RowsData, sqlExecStatus
}

func (sqlExec *SqlExec) Update () error {
    db := SqlCons[sqlExec.TargetDb]

    sqlStmt, err := db.Prepare(sqlExec.Stmt)
    res, err := sqlStmt.Exec()

    if err == nil {
        rowsAffected, _ := res.RowsAffected()
        sqlExec.RowsCount = int(rowsAffected)
    }

    return err
}

func (sqlExec *SqlExec) Delete () error {
    db := SqlCons[sqlExec.TargetDb]

    sqlStmt, err := db.Prepare(sqlExec.Stmt)
    res, err := sqlStmt.Exec()

    if err == nil {
        rowsAffected, _ := res.RowsAffected()
        sqlExec.RowsCount = int(rowsAffected)
    }

    sqlStmt.Close()

    return err
}

func (sqlExec *SqlExec) QueryWithoutParams () error {
    db := SqlCons[sqlExec.TargetDb]

    sqlStmt, err := db.Prepare(sqlExec.Stmt)
    rows, err := sqlStmt.Query()

    if err == nil {
        rowsCount, rowsHeaders, rowsData := ScanRows(rows)

        sqlExec.RowsCount = rowsCount
        sqlExec.RowsHeaders = rowsHeaders
        sqlExec.RowsData = rowsData
    }

    defer rows.Close()
    sqlStmt.Close()

    return err
}

func (sqlExec *SqlExec) QueryWithParams () {

}

func ScanRows (rows *sql.Rows) (int, []string, []map[string]interface{}) {
    rowsHeaders, _ := rows.Columns()
    var rowsData []map[string]interface{}

    scanArgs := make([]interface{}, len(rowsHeaders))
    values := make([]interface{}, len(rowsHeaders))
    for i := range values {
        scanArgs[i] = &values[i]
    }

    rowsCount := 0
    for rows.Next() {
        rows.Scan(scanArgs...)
        record := make(map[string]interface{})

        for i, col := range values {
            if col != nil {
                // note, try best to get the type information to interface{}
                switch col.(type) {
                    case int64:
                        record[rowsHeaders[i]] = col.(int64)
                    case float64:
                        record[rowsHeaders[i]] = col.(float64)
                    default:
                        record[rowsHeaders[i]] = string(col.([]byte))
                }
            }
        }
        rowsCount = rowsCount + 1
        rowsData = append(rowsData, record)
    }

    return rowsCount, rowsHeaders, rowsData
}

func (sqlExec *SqlExec) Insert () error {
    db := SqlCons[sqlExec.TargetDb]

    sqlStmt, err := db.Prepare(sqlExec.Stmt)
    res, err := sqlStmt.Exec()

    if err == nil {
        rowsAffected, _ := res.RowsAffected()
        sqlExec.RowsCount = int(rowsAffected)
    }

    return err
}


