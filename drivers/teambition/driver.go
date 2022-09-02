package teambition

import (
	"context"
	"errors"
	"net/http"

	"github.com/alist-org/alist/v3/drivers/base"
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/go-resty/resty/v2"
)

type Teambition struct {
	model.Storage
	Addition
}

func (d *Teambition) Config() driver.Config {
	return config
}

func (d *Teambition) GetAddition() driver.Additional {
	return d.Addition
}

func (d *Teambition) Init(ctx context.Context, storage model.Storage) error {
	d.Storage = storage
	err := utils.Json.UnmarshalFromString(d.Storage.Addition, &d.Addition)
	if err != nil {
		return err
	}
	_, err = d.request("/api/v2/roles", http.MethodGet, nil, nil)
	return err
}

func (d *Teambition) Drop(ctx context.Context) error {
	return nil
}

func (d *Teambition) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	return d.getFiles(dir.GetID())
}

//func (d *Teambition) Get(ctx context.Context, path string) (model.Obj, error) {
//	// TODO this is optional
//	return nil, errs.NotImplement
//}

func (d *Teambition) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	if u, ok := file.(model.URL); ok {
		url := u.URL()
		res, _ := base.NoRedirectClient.R().Get(url)
		if res.StatusCode() == 302 {
			url = res.Header().Get("location")
		}
		return &model.Link{URL: url}, nil
	}
	return nil, errors.New("can't convert obj to URL")
}

func (d *Teambition) MakeDir(ctx context.Context, parentDir model.Obj, dirName string) error {
	data := base.Json{
		"objectType":     "collection",
		"_projectId":     d.ProjectID,
		"_creatorId":     "",
		"created":        "",
		"updated":        "",
		"title":          dirName,
		"color":          "blue",
		"description":    "",
		"workCount":      0,
		"collectionType": "",
		"recentWorks":    []interface{}{},
		"_parentId":      parentDir.GetID(),
		"subCount":       nil,
	}
	_, err := d.request("/api/collections", http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, nil)
	return err
}

func (d *Teambition) Move(ctx context.Context, srcObj, dstDir model.Obj) error {
	pre := "/api/works/"
	if srcObj.IsDir() {
		pre = "/api/collections/"
	}
	_, err := d.request(pre+srcObj.GetID()+"/move", http.MethodPut, func(req *resty.Request) {
		req.SetBody(base.Json{
			"_parentId": dstDir.GetID(),
		})
	}, nil)
	return err
}

func (d *Teambition) Rename(ctx context.Context, srcObj model.Obj, newName string) error {
	pre := "/api/works/"
	data := base.Json{
		"fileName": newName,
	}
	if srcObj.IsDir() {
		pre = "/api/collections/"
		data = base.Json{
			"title": newName,
		}
	}
	_, err := d.request(pre+srcObj.GetID(), http.MethodPut, func(req *resty.Request) {
		req.SetBody(data)
	}, nil)
	return err
}

func (d *Teambition) Copy(ctx context.Context, srcObj, dstDir model.Obj) error {
	pre := "/api/works/"
	if srcObj.IsDir() {
		pre = "/api/collections/"
	}
	_, err := d.request(pre+srcObj.GetID()+"/fork", http.MethodPut, func(req *resty.Request) {
		req.SetBody(base.Json{
			"_parentId": dstDir.GetID(),
		})
	}, nil)
	return err
}

func (d *Teambition) Remove(ctx context.Context, obj model.Obj) error {
	pre := "/api/works/"
	if obj.IsDir() {
		pre = "/api/collections/"
	}
	_, err := d.request(pre+obj.GetID()+"/archive", http.MethodPost, nil, nil)
	return err
}

func (d *Teambition) Put(ctx context.Context, dstDir model.Obj, stream model.FileStreamer, up driver.UpdateProgress) error {
	res, err := d.request("/projects", http.MethodGet, nil, nil)
	if err != nil {
		return err
	}
	token := GetBetweenStr(string(res), "strikerAuth&quot;:&quot;", "&quot;,&quot;phoneForLogin")
	var newFile *FileUpload
	if stream.GetSize() <= 20971520 {
		// post upload
		newFile, err = d.upload(stream, token)
	} else {
		// chunk upload
		//err = base.ErrNotImplement
		newFile, err = d.chunkUpload(stream, token)
	}
	if err != nil {
		return err
	}
	return d.finishUpload(newFile, dstDir.GetID())
}

func (d *Teambition) Other(ctx context.Context, args model.OtherArgs) (interface{}, error) {
	return nil, errs.NotSupport
}

var _ driver.Driver = (*Teambition)(nil)