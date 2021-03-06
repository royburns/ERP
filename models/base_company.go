package models

import (
	"errors"
	"fmt"
	"github.com/royburns/goERP/utils"
	"strings"
	"time"

	"github.com/astaxie/beego/orm"
)

//Company 公司
type Company struct {
	ID         int64            `orm:"column(id);pk;auto" json:"id" form:"recordID"` //主键
	CreateUser *User            `orm:"rel(fk);null" json:"-"`                        //创建者
	UpdateUser *User            `orm:"rel(fk);null" json:"-"`                        //最后更新者
	CreateDate time.Time        `orm:"auto_now_add;type(datetime)" json:"-"`         //创建时间
	UpdateDate time.Time        `orm:"auto_now;type(datetime)" json:"-"`             //最后更新时间
	Name       string           `orm:"unique" json:"Name" form:"Name"`               //公司名称
	Code       string           `orm:"unique" json:"Code" form:"Code"`               //公司编码
	Children   []*Company       `orm:"reverse(many)" json:"-"`                       //子公司
	Parent     *Company         `orm:"rel(fk);null" json:"-" form:"-"`               //上级公司
	Department []*Department    `orm:"reverse(many)" json:"-"`                       //部门
	Country    *AddressCountry  `orm:"rel(fk);null" json:"-" form:"-"`               //国家
	Province   *AddressProvince `orm:"rel(fk);null" json:"-" form:"-"`               //省份
	City       *AddressCity     `orm:"rel(fk);null" json:"-" form:"-"`               //城市
	District   *AddressDistrict `orm:"rel(fk);null" json:"-" form:"-"`               //区县
	Street     string           `orm:"default()" json:"Street" form:"Street"`        //街道

	ParentID   int64 `orm:"-" json:"Parent" form:"Parent"`     //母公司
	CountryID  int64 `orm:"-" json:"Country" form:"Country"`   //国家
	ProvinceID int64 `orm:"-" json:"Province" form:"Province"` //省份
	CityID     int64 `orm:"-" json:"City" form:"City"`         //城市
	DistrictID int64 `orm:"-" json:"District" form:"District"` //区县
}

func init() {
	orm.RegisterModel(new(Company))
}

// TableName 表名
func (u *Company) TableName() string {
	return "base_company"
}

// AddCompany insert a new Company into database and returns
// last inserted ID on success.
func AddCompany(obj *Company, addUser *User) (id int64, err error) {
	o := orm.NewOrm()
	obj.CreateUser = addUser
	obj.UpdateUser = addUser
	errBegin := o.Begin()
	defer func() {
		if err != nil {
			if errRollback := o.Rollback(); errRollback != nil {
				err = errRollback
			}
		}
	}()
	if errBegin != nil {
		return 0, errBegin
	}
	if obj.ParentID > 0 {
		obj.Parent = new(Company)
		obj.Parent.ID = obj.ParentID
	}
	if obj.CountryID > 0 {
		obj.Country = new(AddressCountry)
		obj.Country.ID = obj.CountryID
	}
	if obj.ProvinceID > 0 {
		obj.Province = new(AddressProvince)
		obj.Province.ID = obj.ProvinceID
	}
	if obj.CityID > 0 {
		obj.City = new(AddressCity)
		obj.City.ID = obj.CityID
	}
	if obj.DistrictID > 0 {
		obj.District = new(AddressDistrict)
		obj.District.ID = obj.DistrictID
	}
	id, err = o.Insert(obj)
	if err == nil {
		errCommit := o.Commit()
		if errCommit != nil {
			return 0, errCommit
		}
	}
	return id, err
}

// GetCompanyByID retrieves Company by ID. Returns error if
// ID doesn't exist
func GetCompanyByID(id int64) (obj *Company, err error) {
	o := orm.NewOrm()
	obj = &Company{ID: id}
	if err = o.Read(obj); err == nil {
		o.LoadRelated(obj, "Department")
		o.LoadRelated(obj, "Children")
		o.LoadRelated(obj, "Parent")
		o.LoadRelated(obj, "Country")
		o.LoadRelated(obj, "Province")
		o.LoadRelated(obj, "City")
		o.LoadRelated(obj, "District")
		return obj, err
	}
	return nil, err
}

// GetCompanyByName retrieves Company by Name. Returns error if
// Name doesn't exist
func GetCompanyByName(name string) (*Company, error) {
	o := orm.NewOrm()
	var obj Company
	cond := orm.NewCondition()
	cond = cond.And("Name", name)
	qs := o.QueryTable(&obj)
	qs = qs.SetCond(cond)
	err := qs.One(&obj)
	return &obj, err
}

// GetAllCompany retrieves all Company matches certain condition. Returns empty list if
// no records exist
func GetAllCompany(query map[string]interface{}, exclude map[string]interface{}, condMap map[string]map[string]interface{}, fields []string, sortby []string, order []string, offset int64, limit int64) (utils.Paginator, []Company, error) {
	var (
		objArrs   []Company
		paginator utils.Paginator
		num       int64
		err       error
	)
	if limit == 0 {
		limit = 20
	}

	o := orm.NewOrm()
	qs := o.QueryTable(new(Company))
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
			}
		}
	}
	return paginator, objArrs, err
}

// UpdateCompany updates Company by ID and returns error if
// the record to be updated doesn't exist
func UpdateCompany(obj *Company, updateUser *User) (id int64, err error) {
	o := orm.NewOrm()
	obj.UpdateUser = updateUser
	updateFields := []string{"UpdateUser", "UpdateDate", "Name", "Code", "Parent", "Country", "Province", "City", "District", "Street"}
	if obj.ParentID > 0 {
		obj.Parent = new(Company)
		obj.Parent.ID = obj.ParentID
	}
	if obj.CountryID > 0 {
		obj.Country = new(AddressCountry)
		obj.Country.ID = obj.CountryID
	}
	if obj.ProvinceID > 0 {
		obj.Province = new(AddressProvince)
		obj.Province.ID = obj.ProvinceID
	}
	if obj.CityID > 0 {
		obj.City = new(AddressCity)
		obj.City.ID = obj.CityID
	}
	if obj.DistrictID > 0 {
		obj.District = new(AddressDistrict)
		obj.District.ID = obj.DistrictID
	}
	_, err = o.Update(obj, updateFields...)
	return obj.ID, err
}

// DeleteCompany deletes Company by ID and returns error if
// the record to be deleted doesn't exist
func DeleteCompany(id int64) (err error) {
	o := orm.NewOrm()
	v := Company{ID: id}
	// ascertain id exists in the database
	if err = o.Read(&v); err == nil {
		var num int64
		if num, err = o.Delete(&Company{ID: id}); err == nil {
			fmt.Println("Number of records deleted in database:", num)
		}
	}
	return
}
