package main

import (
	"bufio"
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
	"sort"
)

type VariantType string
type VariantData string

type EnumVariants map[VariantType]VariantData

type EnumInfo struct {
	Constraint string       `toml:"constraint"`
	Variants   EnumVariants `toml:"variants"`
}
type EnumTypeMap map[string]EnumInfo

func nl(v string) string {
	v += "\n"
	return v
}

func generateCode(m EnumTypeMap) string {
	var code string
	var sorted_list = sort.StringSlice{}
	for enum_name := range m {
		sorted_list = append(sorted_list, enum_name)
	}
	sorted_list.Sort()

	var json_required bool
	for _, enum_name := range sorted_list {
		var enum_info = m[enum_name]

		var sorted_var_list = sort.StringSlice{}
		for v := range enum_info.Variants {
			sorted_var_list = append(sorted_var_list, string(v))
		}
		sorted_var_list.Sort()

		code += nl(fmt.Sprintf("type %s struct { \n Type %s `json:\"type\"` \n Data interface{} `json:\"data\"` \n }",
			enum_name,
			enum_info.Constraint))
		var unmarshal_code = func() *string {
			var case_code = func() *string {
				var case_code string
				var sorted_list = sort.StringSlice{}
				for v, _ := range enum_info.Variants {
					sorted_list = append(sorted_list, string(v))
				}
				sorted_list.Sort()
				for _, var_name := range sorted_list {
					case_code += nl(fmt.Sprintf("case %s:", var_name))
					var assoc_type = enum_info.Variants[VariantType(var_name)]
					if assoc_type != "null" {
						case_code += nl(fmt.Sprintf("if !data_found { return errors.New(\"No associated data found for enum %s\") }", enum_name))
						case_code += nl(fmt.Sprintf("	var d %s", enum_info.Variants[VariantType(var_name)]))
						case_code += nl("	var data_err = json.Unmarshal(data_raw, &d)")
						case_code += nl("       if data_err != nil { return data_err }")
						case_code += nl("self.Data = &d")
					} else {
						case_code += nl("break")
					}
				}

				return &case_code
			}()

			if case_code == nil {
				return nil
			}
			var v string

			v += nl("var doc map[string]json.RawMessage")
			v += nl("if err := json.Unmarshal(b, &doc); err != nil { return err }")
			v += nl("if doc == nil { return nil }")
			v += nl(`var t_raw, t_found = doc["type"]; if !t_found { return nil }`)
			v += nl(`var data_raw, data_found = doc["data"]; if bytes.Equal(data_raw, []byte("null")) { data_found = false }`)
			v += nl(fmt.Sprintf(`var t %s`, enum_info.Constraint))
			v += nl(`if t_err := json.Unmarshal(t_raw, &t); t_err != nil { return t_err }`)

			v += nl(`switch t.Value().(type) {`)
			v += nl(*case_code)
			v += nl("}")
			v += nl(`self.Type = t`)

			v += nl(`return nil`)
			return &v
		}()
		if unmarshal_code != nil {
			json_required = true
			code += nl(fmt.Sprintf("func (self *%s) UnmarshalJSON(b []byte) error {\n %s \n}\n", enum_name, *unmarshal_code))
		}
	}

	var imports string
	if len(sorted_list) > 0 {
		imports += nl(`import (`)
		imports += nl(`"errors"`)
		if json_required {
			imports += nl(`"bytes"`)
			imports += nl(`"encoding/json"`)
		}
		imports += nl(`)`)
	}

	return imports + code
}

func main() {
	var data string
	var scanner = bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		data += scanner.Text()
		data += "\n"
	}

	var m = EnumTypeMap{}
	var err = toml.Unmarshal([]byte(data), &m)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println(generateCode(m))
}
