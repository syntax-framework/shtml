package jsc

import (
	"github.com/iancoleman/strcase"
	"github.com/syntax-framework/shtml/cmn"
	"github.com/syntax-framework/shtml/sht"
	"strings"
)

var errorCompClientParamReferenceNotFound = sht.Err(
	"component.param.client.ref.notfound",
	"The referenced parameter does not exist.", `Param: client-param-%s="@%s"`, "Component: %s",
)

var errorCompClientParamInvalidName = sht.Err(
	"component.param.client.name",
	"The parameter name is invalid.", `Param: client-param-%s="@%s"`, "Component: %s",
)

// ParseComponentParams Processes the parameters of a component
//
// Qualquer componente
func ParseComponentParams(node *sht.Node) (*cmn.NodeComponentParams, error) {
	// parse attributes
	var serverParams []cmn.ComponentParam // server params
	var clientParams []cmn.ComponentParam // javascript params
	var attrsToRemove []*sht.Attribute
	serverParamsByName := map[string]*cmn.ComponentParam{}
	clientParamsByName := map[string]*cmn.ComponentParam{}

	refParamsValueOrig := map[string]string{}
	clientParamsToResolve := map[string]*cmn.ComponentParam{}

	for name, attr := range node.Attributes.Map {
		isParam, isClientParam, paramName := strings.HasPrefix(name, "param-"), false, ""
		if isParam {
			paramName = strcase.ToLowerCamel(strings.Replace(name, "param-", "", 1))
		} else {
			isClientParam = strings.HasPrefix(name, "client-param-")
			if isClientParam {
				paramName = strcase.ToLowerCamel(strings.Replace(name, "client-param-", "", 1))
			}
		}

		if isParam || isClientParam {
			if paramName != "" {
				param := cmn.ComponentParam{
					Name:     paramName,
					Required: true,
					IsClient: isClientParam,
				}
				paramTypeName := strings.TrimSpace(attr.Value)
				if strings.HasPrefix(paramTypeName, "?") {
					param.Required = false
					paramTypeName = paramTypeName[1:]
				}

				if isClientParam && strings.HasPrefix(paramTypeName, "@") {
					// is exposing a parameter to JS, by reference
					// Ex. <component param-name="string" client-param-name="@name" />
					referenceName := strcase.ToLowerCamel(paramTypeName[1:])
					refParamsValueOrig[referenceName] = paramTypeName[1:]

					serverParam, serverParamFound := serverParamsByName[referenceName]
					if serverParamFound {
						param.Type = serverParam.Type
						param.TypeName = serverParam.TypeName
						param.Reference = serverParam
					} else {
						// will solve further below
						clientParamsToResolve[referenceName] = &param
					}
				} else {
					paramType, paramTypeFound := cmn.ParamTypeNames[paramTypeName]
					if !paramTypeFound {
						paramType = cmn.ParamTypeUnknown
					}

					param.Type = paramType
					param.TypeName = paramTypeName
				}

				if isClientParam {
					// @TODO: Valid param types (string, number, bool, array, object)
					// param name is valid?
					if _, isInvalid := ClientInvalidParamsAndRefs[paramName]; isInvalid || strings.HasPrefix(paramName, "_$") {
						return nil, errorCompClientParamInvalidName(strcase.ToKebab(paramName), node.DebugTag())
					}

					clientParams = append(clientParams, param)
					clientParamsByName[paramName] = &param
				} else {
					serverParams = append(serverParams, param)
					serverParamsByName[paramName] = &param
				}
			}

			attrsToRemove = append(attrsToRemove, attr)
		}
	}

	// resolve ClientParams reference
	for referenceName, jsParam := range clientParamsToResolve {
		serverParam, serverParamFound := serverParamsByName[referenceName]
		if serverParamFound {
			jsParam.Type = serverParam.Type
			jsParam.TypeName = serverParam.TypeName
			jsParam.Reference = serverParam
		} else {
			// Error, is referencing a non-existent parameter
			return nil, errorCompClientParamReferenceNotFound(
				strcase.ToKebab(jsParam.Name), strcase.ToKebab(refParamsValueOrig[referenceName]), node.DebugTag(),
			)
		}
	}

	return &cmn.NodeComponentParams{
		ServerParams:       serverParams,
		ClientParams:       clientParams,
		AttrsToRemove:      attrsToRemove,
		ServerParamsByName: serverParamsByName,
		ClientParamsByName: clientParamsByName,
	}, nil
}
