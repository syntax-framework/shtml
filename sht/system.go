package sht

type TemplateSystem struct {
	Loader     func(filepath string) (string, error)
	Directives *Directives
}

// Load faz o carreagmento de um arquivo html
func (s *TemplateSystem) Load(filepath string) (string, error) {
	var content, err = s.Loader(filepath)
	if err != nil {
		return "", err
	}

	// Debug information
	//var line = 1
	//transcludeSlots = ("\n<!--L:1 " + filepath + "-->") + regLF.ReplaceAllStringFunc(transcludeSlots, func(s string) string {
	//	line++
	//	return "\n<!--L:" + strconv.Itoa(line) + " " + filepath + "-->"
	//})
	return content, nil
}

func (s *TemplateSystem) Compile(filepath string) (*Compiled, error) {
	content, err := s.Load(filepath)
	if err != nil {
		return nil, err
	}

	compiler := NewCompiler(s)
	compiler.SetFilepath(filepath)

	return compiler.Compile(content)
}
