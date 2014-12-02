package jsph

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/robertkrimen/otto"
)

var js *otto.Otto = otto.New()

func getFileContentsAsString(path string) (string, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func CompileJsphFile(path string) func(interface{}) string {
	templateString, _ := getFileContentsAsString(path)
	templateString = "?>" + templateString + "<?"
	// if ! templateString ... boom!
	inHtml, _ := regexp.Compile("[\\?\\%]>=?[\\s\\S]*?<[\\?\\%]")
	inJs, _ := regexp.Compile("<[\\?\\%]=?[\\s\\S]*?<?[\\?\\%]>")
	htmlMatches := inHtml.FindAllString(templateString, -1)
	htmlMatchIndexes := inHtml.FindAllStringSubmatchIndex(templateString, -1)
	jsMatches := inJs.FindAllString(templateString, -1)
	lastHtmlMatchEndedAt := len(templateString)

	functionBody :=
		"var templates = templates || {} \n" +
			"templates[" + toQuotedJsStringLiteral(path) + "] = function(vars) { \n" +
			"  return (function() { \n" +
			"    var o = \"\"; \n"

	for i := 0; i < len(htmlMatches) || i < len(jsMatches); i++ {
		if i < len(htmlMatches) {
			htmlMatch := htmlMatches[i][2 : len(htmlMatches[i])-2]
			if len(htmlMatch) > 0 {
				functionBody += "    o += " + toQuotedJsStringLiteral(htmlMatch) + ";\n"
			}
			lastHtmlMatchEndedAt = htmlMatchIndexes[i][0] + len(htmlMatches[i])
		}
		if i < len(jsMatches) {
			jsMatch := jsMatches[i][2 : len(jsMatches[i])-2]
			if len(jsMatch) > 0 {
				if jsMatch[0] == '=' {
					functionBody += "    o += (" + jsMatch[1:] + ");\n"
				} else {
					functionBody += "    " + jsMatch + "\n"
				}
			}

		} else if lastHtmlMatchEndedAt < len(templateString) {
			//they left off the closing tag, treat the rest like a js script
			jsMatch := templateString[lastHtmlMatchEndedAt : len(templateString)-2]
			functionBody += "    " + jsMatch + "\n"
		}
	}

	functionBody +=
		"    return o; \n" +
			"  }).call(vars); \n" +
			"} \n"

	_, err := js.Run(functionBody)
	if err != nil {
		fmt.Println(functionBody)
		fmt.Println("%s", err)
	}

	renderFunction := func(stuff interface{}) string {
		jsonValues, _ := json.Marshal(stuff)
		result, _ := js.Run("templates[" + toQuotedJsStringLiteral(path) + "](" + string(jsonValues) + ")")
		resultString, _ := result.ToString()
		return resultString
	}

	return renderFunction
}

func toQuotedJsStringLiteral(str string) string {
	str = strings.Replace(str, "\\", "\\\\", -1)
	str = strings.Replace(str, "\"", "\\\"", -1)
	str = strings.Replace(str, "'", "\\'", -1)
	str = strings.Replace(str, "\n", "\\n", -1)
	str = strings.Replace(str, "\r", "\\r", -1)
	str = strings.Replace(str, "\t", "\\t", -1)
	return "\"" + str + "\""
}
