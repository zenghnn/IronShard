package IronShard

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Shard struct {
	DB           *gorm.DB
	DBName       string
	TbPrefix     string
	fieldSql     string
	needMerge    bool
	MaxId        int64
	LastTableIdx int
	TableIdxs    []int
	TableNames   []string
	umAddTbLock  *sync.RWMutex
}
type MysqlSchema struct {
	Name      string `gorm:"column:TABLE_NAME;type:timestamp;" json:"name"`
	AutoIncre int64  `gorm:"column:AUTO_INCREMENT;type:bigint;" json:"auto_increment"`
}

//初始化碎片系统包括redis缓存
func NewShard(DB *gorm.DB, dbname string) Shard {
	return Shard{DB: DB, DBName: dbname}
}

const TableCountLimit = 100000
const TableCountLimitStr = "100000"

// @Title Init
// @Description   初始化分表一般在启动服务时
// @Param tbPrefix  "表的名称，分表后作为前缀"
// @Param structSql  表结构sql，目前只支持sql
func (sm *Shard) Init(tbPrefix string, structSql string, needMerge bool, priKey string) (ts []MysqlSchema, err error) {
	sm.fieldSql = structSql
	tableIdx := 1
	tableIdxstr := strconv.Itoa(tableIdx)
	curIdxTable := tbPrefix + "_" + tableIdxstr
	sm.TbPrefix = tbPrefix
	sm.needMerge = needMerge
	count := sm.DB.Raw("select * from information_schema.TABLES where TABLE_NAME REGEXP '^" + tbPrefix + "_[0-9]{1,6}$' and TABLE_SCHEMA='" + sm.DBName + "' ").Scan(&ts).RowsAffected
	if count == 0 {
		sql1 := "CREATE TABLE `" + curIdxTable + "` (" + structSql
		if needMerge {
			sql1 += "  ) ENGINE=MYISAM AUTO_INCREMENT=" + TableCountLimitStr + " DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;"
		} else {
			sql1 += "  ) ENGINE=InnoDB AUTO_INCREMENT=" + TableCountLimitStr + " DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;"
		}
		err = sm.DB.Exec(sql1).Error
		if needMerge {
			sql2 := "CREATE TABLE `" + tbPrefix + "_merge` (" + structSql +
				"  ) ENGINE=MERGE DEFAULT CHARSET=utf8mb4 CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci UNION = (" + curIdxTable + ");"
			err = sm.DB.Exec(sql2).Error
			if err != nil {
				err = errors.New("create user_m_merge error" + err.Error())
				return
			}
		}
		sm.TableIdxs = []int{tableIdx}
		sm.LastTableIdx = tableIdx
		sm.TableNames = append(sm.TableNames, curIdxTable)
		sm.MaxId = TableCountLimit - 1
		//TODAY_TABLE_NAME = curIdxTable
	} else {
		if !needMerge {
			sm.TableIdxs = []int{}
			sm.LastTableIdx = tableIdx
		}

		lengthint := len(ts)
		tables := ""
		sort.Slice(ts, func(i, j int) bool {
			arrA := strings.Split(ts[i].Name, "_")
			arrB := strings.Split(ts[j].Name, "_")
			a, _ := strconv.Atoi(arrA[2])
			b, _ := strconv.Atoi(arrB[2])
			return a < b
		})
		lastUm := ts[lengthint-1].Name
		for k2, i2 := range ts {
			if k2 > 0 && k2 < lengthint {
				tables += ","
			}
			tables += i2.Name

			splitTname := strings.Split(i2.Name, "_")
			loctbidx, _ := strconv.Atoi(splitTname[2])
			sm.TableIdxs = append(sm.TableIdxs, loctbidx)
			sm.TableNames = append(sm.TableNames, i2.Name)
		}
		splitUm := strings.Split(lastUm, "_")
		lastUmCount, _ := strconv.Atoi(splitUm[2])
		curIdxTable = tbPrefix + "_" + strconv.Itoa(lastUmCount)

		if needMerge {
			err = sm.DB.Exec("ALTER TABLE " + tbPrefix + "_merge UNION = (" + tables + ")").Error
			if err != nil {
				errors.New("ALTER user_m_merge tables error :" + err.Error())
				fmt.Println("ALTER user_m_merge tables error :" + err.Error())
			}
		}

		sm.LastTableIdx = lastUmCount
		maxidStruct := SelecMaxId{}
		count = sm.DB.Raw("select " + priKey + " as maxid from " + lastUm + " order by " + priKey + " desc ").First(&maxidStruct).RowsAffected
		if err != nil {
			fmt.Println("err--select max(id) as maxid from--", err.Error())
			err = errors.New("select max(id) as maxid from " + lastUm + err.Error())
			return
		}
		if count > 0 {
			sm.MaxId = maxidStruct.Maxid
		} else {
			sm.MaxId = TableCountLimit - 1
		}
		sm.umAddTbLock = new(sync.RWMutex)
	}
	return
}

type SelecMaxId struct {
	Maxid int64 `gorm:"column:maxid;type:bigint;"`
}

//新增一个主表 并刷新最新表名，最后一个表的idx等信息
func (sm *Shard) NewTable() error {
	sm.umAddTbLock.Lock()
	tableIdx := sm.LastTableIdx + 1
	curIdxTable := sm.TbPrefix + "_" + strconv.Itoa(tableIdx)
	sql1 := "CREATE TABLE `" + curIdxTable + "` (" + sm.fieldSql
	locAutoIncrement := int64(TableCountLimit) * int64(sm.LastTableIdx)
	locAutoIncrementStr := strconv.FormatInt(locAutoIncrement, 10)
	if sm.needMerge {
		sql1 += "  ) ENGINE=MYISAM AUTO_INCREMENT=" + locAutoIncrementStr + " DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;"
	} else {
		sql1 += "  ) ENGINE=InnoDB AUTO_INCREMENT=" + locAutoIncrementStr + " DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;"
	}
	err := sm.DB.Exec(sql1).Error
	if err != nil {
		return err
	}
	if sm.needMerge {
		sql2 := "CREATE TABLE `" + sm.TbPrefix + "_merge` (" + sm.fieldSql +
			"  ) ENGINE=MERGE DEFAULT CHARSET=utf8mb4 CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci UNION = (" + curIdxTable + ");"
		err = sm.DB.Exec(sql2).Error
		if err != nil {
			err = errors.New("create user_m_merge error" + err.Error())
			return err
		}
	}
	sm.TableIdxs = append(sm.TableIdxs, tableIdx)
	sm.LastTableIdx = tableIdx
	sm.TableNames = append(sm.TableNames, curIdxTable)
	sm.umAddTbLock.Unlock()
	return nil
}
