package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

const (
	ActionDelete  = "delete"
	ActionReplace = "replace"
	ActionMerge   = "merge"
	ActionAppend  = "append"
)

type Commands struct {
	Commands []Command `yaml:"commands"`
}

type Command struct {
	Path   string            `yaml:"path"`
	Fields map[string]string `yaml:"has_fields"`
	Action string            `yaml:"action"`
	Params interface{}       `yaml:"params"`
}

func main() {
	app := &cli.App{
		Name:  "ayed",
		Usage: "Another YAML editor",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "f",
				Value: "",
				Usage: "yaml file",
			},
			&cli.StringFlag{
				Name:  "s",
				Value: "",
				Usage: "script file",
			},
		},
		Action: func(c *cli.Context) error {
			App(c.String("s"), c.String("f"))
			return nil
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

// App run application
func App(scriptFile string, destFile string) {
	var commands = &Commands{}
	if scriptFile != "" {
		file, err := os.Open(scriptFile)
		check(err)
		var raw = yaml.MapSlice{}
		err = yaml.NewDecoder(file).Decode(&raw)
		check(err)
		_ = file.Close()
		data, _ := yaml.Marshal(raw)
		err = yaml.Unmarshal(data, commands)
		check(err)
		for _, pair := range raw {
			if pair.Key.(string) == "commands" {
				list := pair.Value.([]interface{})
				for i, cmd := range list {
					commands.Commands[i].Params = YamlMapGet(cmd.(yaml.MapSlice), "params")

				}
			}
		}
	} else {

	}
	var file io.ReadCloser
	if destFile == "" {
		file = os.Stdin
	} else {
		var err error
		file, err = os.Open(destFile)
		check(err)
	}

	var object = yaml.MapSlice{}
	err := yaml.NewDecoder(file).Decode(&object)
	check(err)
	_ = file.Close()
	defer func() {
		err := recover()
		if err != nil {
			log.Fatalln(err)
		}
	}()
	for _, cmd := range commands.Commands {
		ctx := &Context{
			Command:    &cmd,
			ParentPath: "",
		}
		WalkAllNode(ctx, "", object, func(v interface{}) {
			object = v.(yaml.MapSlice)
		}, func() {
		})
	}

	data, err := yaml.Marshal(object)
	check(err)
	fmt.Print(string(data))
}

type Context struct {
	*Command
	ParentPath string
}

func escape(str string) string {
	str = strings.Replace(str, "\\", "\\\\", -1)
	return strings.Replace(str, ".", "\\.", -1)
}

func regMatch(reg string, text string) bool {
	if reg[0] != '^' {
		reg = "^" + reg
	}
	if reg[len(reg)-1] != '$' {
		reg = reg + "$"
	}
	return regexp.MustCompile(reg).MatchString(text)
}

func mergeIn(m yaml.MapSlice, p yaml.MapSlice) yaml.MapSlice {
	for _, pkv := range p {
		has := false
		for mi, mkv := range m {
			if reflect.DeepEqual(mkv.Key, pkv.Key) {
				has = true
				mv, mok := mkv.Value.(yaml.MapSlice)
				pv, pok := pkv.Value.(yaml.MapSlice)
				if mok && pok {
					m[mi] = yaml.MapItem{Key: mkv.Key, Value: mergeIn(mv, pv)}
				} else {
					m[mi] = pkv
				}
			}
		}
		if !has {
			m = append(m, pkv)
		}
	}
	return m
}

func matchKV(kv map[string]string, hash yaml.MapSlice) bool {
	trans := map[string]string{}
	for _, pair := range hash {
		switch pair.Value.(type) {
		case yaml.MapSlice:
		case []interface{}:
		default:
			trans[fmt.Sprint(pair.Key)] = fmt.Sprint(pair.Value)
		}
	}
	for k, v := range kv {
		if val,ok:=trans[k];!ok{
			return false
		}else if !regMatch(v, val){
			return false
		}
	}
	return true
}

func doAction(cmd *Command, object interface{}, action string, params interface{}, remove func()) (nObject interface{}, skip bool) {
	switch action {
	case ActionAppend:
		switch realObject := object.(type) {
		case []interface{}:
			realObject = append(realObject, params)
			return realObject, false
		default:
			panic("not support append on non-array node")
		}
	case ActionMerge:
		switch realObject := object.(type) {
		case yaml.MapSlice:
			trans, ok := params.(yaml.MapSlice)
			if !ok {
				panic("not support merge non-map node into map node")
			}
			realObject = mergeIn(realObject, trans)
			return realObject, false
		case []interface{}:
			params, ok := params.([]interface{})
			if !ok {
				panic("not support merge non-array node into array node")
			}
			realObject = append(realObject, params...)
			return realObject, false
		default:
			panic("not support merge into scalar node")
		}
	case ActionDelete:
		remove()
		return object, true
	case ActionReplace:
		return params, false
	default:
		panic("not support such action: " + action)
	}
}

func Filter(beforeLoop bool, currentPath string, key interface{}, object interface{}, ctx *Context, remove func()) (nObject interface{}, skip bool) {
	if beforeLoop && ctx.Action != ActionDelete {
		return object, false
	}
	if !beforeLoop && ctx.Action == ActionDelete {
		return object, false
	}
	if ctx.Path != "" && !regMatch(ctx.Path, currentPath) {
		return object, false
	}
	if len(ctx.Fields) != 0 {
		values, ok := object.(yaml.MapSlice)
		if !ok {
			return object, false
		}
		if ctx.Path != "" && !regMatch(ctx.Path, currentPath) {
			return object, false
		}
		if !matchKV(ctx.Fields, values) {
			return object, false
		}
	}
	return doAction(ctx.Command, object, ctx.Action, ctx.Params, remove)
}

func WalkAllNode(ctx *Context, key interface{}, object interface{}, setter func(v interface{}), remove func()) {
	var currentPath string
	if key != "" {
		if ctx.ParentPath == "" {
			currentPath = escape(fmt.Sprint(key))
		} else {
			currentPath = ctx.ParentPath + "." + escape(fmt.Sprint(key))
		}
	}
	object, skip := Filter(true, currentPath, key, object, ctx, remove)
	if skip {
		return
	}
	switch value := object.(type) {
	case yaml.MapSlice:
		var copied yaml.MapSlice
		for _, pair := range value {
			var remove = false
			var parent = ctx.ParentPath
			ctx.ParentPath = currentPath

			WalkAllNode(ctx, pair.Key, pair.Value, func(v interface{}) {
				pair.Value = v
			}, func() {
				remove = true
			})
			if !remove {
				copied = append(copied, pair)
			}
			ctx.ParentPath = parent
		}
		setter(copied)
		object = copied
	case []interface{}:
		var copied []interface{}
		for k, val := range value {
			var remove = false
			var parent = ctx.ParentPath
			ctx.ParentPath = currentPath
			WalkAllNode(ctx, k, val, func(v interface{}) {
				val = v
			}, func() {
				remove = true
			})
			if !remove {
				copied = append(copied, val)
			}
			ctx.ParentPath = parent
		}
		setter(copied)
		object = copied
	default:
	}
	object, _ = Filter(false, currentPath, key, object, ctx, remove)
	setter(object)
}

func check(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func YamlMapGet(m yaml.MapSlice, key string) (value interface{}) {
	for _, pair := range m {
		if fmt.Sprint(pair.Key) == key {
			return pair.Value
		}
	}
	return value
}
