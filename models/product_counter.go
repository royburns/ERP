package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"goERP/utils"

	"github.com/astaxie/beego/orm"
)

// ProductCounter 柜台
type ProductCounter struct {
	ID               int64              `orm:"column(id);pk;auto" json:"id"`         //主键
	CreateUser       *User              `orm:"rel(fk);null" json:"-"`                //创建者
	UpdateUser       *User              `orm:"rel(fk);null" json:"-"`                //最后更新者
	CreateDate       time.Time          `orm:"auto_now_add;type(datetime)" json:"-"` //创建时间
	UpdateDate       time.Time          `orm:"auto_now;type(datetime)" json:"-"`     //最后更新时间
	Name             string             `orm:"unique"  json:"Name"`                  //产品属性名称
	Description      string             `orm:"type(text);null"  json:"Description"`  //描述
	ProductProducts  []*ProductProduct  `orm:"reverse(many)"`                        //产品规格
	ProductTemplates []*ProductTemplate `orm:"reverse(many)"`                        //产品款式
	ProductsCount    int                `orm:"-"`                                    //产品规格数量
	TemplatesCount   int                `orm:"-"`                                    //产品款式数量

	// form表单使用字段
	FormAction string `orm:"-" json:"FormAction"` //非数据库字段，用于表示记录的增加，修改

}

func init() {
	orm.RegisterModel(new(ProductCounter))
}

// AddProductCounter insert a new ProductCounter into database and returns
// last inserted ID on success.
func AddProductCounter(obj *ProductCounter, addUser *User) (id int64, errs []error) {
	o := orm.NewOrm()
	obj.CreateUser = addUser
	obj.UpdateUser = addUser
	var err error
	err = o.Begin()
	if err != nil {
		errs = append(errs, err)
	}

	id, err = o.Insert(obj)
	if err != nil {
		errs = append(errs, err)
		err = o.Rollback()
		if err != nil {
			errs = append(errs, err)
		}
	} else {
		err = o.Commit()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return id, errs
}

// GetProductCounterByID retrieves ProductCounter by ID. Returns error if
// ID doesn't exist
func GetProductCounterByID(id int64) (obj *ProductCounter, err error) {
	o := orm.NewOrm()
	obj = &ProductCounter{ID: id}
	if err = o.Read(obj); err == nil {
		return obj, nil
	}
	return nil, err
}

// GetAllProductCounter retrieves all ProductCounter matches certain condition. Returns empty list if
// no records exist
func GetAllProductCounter(query map[string]interface{}, exclude map[string]interface{}, condMap map[string]map[string]interface{}, fields []string, sortby []string, order []string, offset int64, limit int64) (utils.Paginator, []ProductCounter, error) {
	var (
		objArrs   []ProductCounter
		paginator utils.Paginator
		num       int64
		err       error
	)
	if limit == 0 {
		limit = 20
	}
	o := orm.NewOrm()
	qs := o.QueryTable(new(ProductCounter))
	qs = qs.RelatedSel()

	//cond k=v cond必须放到Filter和Exclude前面
	cond := orm.NewCondition()
	if _, ok := condMap["and"]; ok {
		andMap := condMap["and"]
		for k, v := range andMap {
			k = strings.Replace(k, ".", "__", -1)
			cond = cond.And(k, v)
		}
	}
	if _, ok := condMap["or"]; ok {
		orMap := condMap["or"]
		for k, v := range orMap {
			k = strings.Replace(k, ".", "__", -1)
			cond = cond.Or(k, v)
		}
	}
	qs = qs.SetCond(cond)
	// query k=v
	for k, v := range query {
		// rewrite dot-notation to Object__Attribute
		k = strings.Replace(k, ".", "__", -1)
		qs = qs.Filter(k, v)
	}
	//exclude k=v
	for k, v := range exclude {
		// rewrite dot-notation to Object__Attribute
		k = strings.Replace(k, ".", "__", -1)
		qs = qs.Exclude(k, v)
	}

	// order by:
	var sortFields []string
	if len(sortby) != 0 {
		if len(sortby) == len(order) {
			// 1) for each sort field, there is an associated order
			for i, v := range sortby {
				orderby := ""
				if order[i] == "desc" {
					orderby = "-" + strings.Replace(v, ".", "__", -1)
				} else if order[i] == "asc" {
					orderby = strings.Replace(v, ".", "__", -1)
				} else {
					return paginator, nil, errors.New("Error: Invalid order. Must be either [asc|desc]")
				}
				sortFields = append(sortFields, orderby)
			}
			qs = qs.OrderBy(sortFields...)
		} else if len(sortby) != len(order) && len(order) == 1 {
			// 2) there is exactly one order, all the sorted fields will be sorted by this order
			for _, v := range sortby {
				orderby := ""
				if order[0] == "desc" {
					orderby = "-" + strings.Replace(v, ".", "__", -1)
				} else if order[0] == "asc" {
					orderby = strings.Replace(v, ".", "__", -1)
				} else {
					return paginator, nil, errors.New("Error: Invalid order. Must be either [asc|desc]")
				}
				sortFields = append(sortFields, orderby)
			}
		} else if len(sortby) != len(order) && len(order) != 1 {
			return paginator, nil, errors.New("Error: 'sortby', 'order' sizes mismatch or 'order' size is not 1")
		}
	} else {
		if len(order) != 0 {
			return paginator, nil, errors.New("Error: unused 'order' fields")
		}
	}

	qs = qs.OrderBy(sortFields...)
	if cnt, err := qs.Count(); err == nil {
		if cnt > 0 {
			paginator = utils.GenPaginator(limit, offset, cnt)
			if num, err = qs.Limit(limit, offset).All(&objArrs, fields...); err == nil {
				paginator.CurrentPageSize = num
				for i, _ := range objArrs {
					o.LoadRelated(&objArrs[i], "ProductProducts")
					objArrs[i].ProductsCount = len(objArrs[i].ProductProducts)
					o.LoadRelated(&objArrs[i], "ProductTemplates")
					objArrs[i].TemplatesCount = len(objArrs[i].ProductTemplates)
				}
			}
		}
	}

	return paginator, objArrs, err
}

// UpdateProductCounterByID updates ProductCounter by ID and returns error if
// the record to be updated doesn't exist
func UpdateProductCounterByID(m *ProductCounter) (err error) {
	o := orm.NewOrm()
	v := ProductCounter{ID: m.ID}
	// ascertain id exists in the database
	if err = o.Read(&v); err == nil {
		var num int64
		if num, err = o.Update(m); err == nil {
			fmt.Println("Number of records updated in database:", num)
		}
	}
	return
}

// GetProductCounterByName retrieves ProductCounter by Name. Returns error if
// Name doesn't exist
func GetProductCounterByName(name string) (obj *ProductCounter, err error) {
	o := orm.NewOrm()
	obj = &ProductCounter{Name: name}
	if err = o.Read(obj); err == nil {
		return obj, nil
	}
	return nil, err
}

// DeleteProductCounter deletes ProductCounter by ID and returns error if
// the record to be deleted doesn't exist
func DeleteProductCounter(id int64) (err error) {
	o := orm.NewOrm()
	v := ProductCounter{ID: id}
	// ascertain id exists in the database
	if err = o.Read(&v); err == nil {
		var num int64
		if num, err = o.Delete(&ProductCounter{ID: id}); err == nil {
			fmt.Println("Number of records deleted in database:", num)
		}
	}
	return
}