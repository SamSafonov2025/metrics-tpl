package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// StructInfo содержит информацию о структуре для генерации
type StructInfo struct {
	Name    string
	Fields  []FieldInfo
	PkgName string
	PkgPath string
}

// FieldInfo содержит информацию о поле структуры
type FieldInfo struct {
	Name      string
	Type      string
	IsPointer bool
	IsSlice   bool
	IsMap     bool
	IsStruct  bool
}

func main() {
	// Получаем корневую директорию проекта
	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Собираем информацию о структурах с комментарием // generate:reset
	structsByPackage := make(map[string][]StructInfo)

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Пропускаем директории vendor, .git и сгенерированные файлы
		if info.IsDir() {
			if info.Name() == "vendor" || info.Name() == ".git" || info.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		// Обрабатываем только .go файлы, кроме тестов и уже сгенерированных
		if !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") ||
			strings.HasSuffix(path, ".gen.go") {
			return nil
		}

		// Парсим файл
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			log.Printf("Failed to parse %s: %v", path, err)
			return nil
		}

		// Ищем структуры с комментарием // generate:reset
		structs := findResetableStructs(file, path)
		if len(structs) > 0 {
			pkgPath := filepath.Dir(path)
			structsByPackage[pkgPath] = append(structsByPackage[pkgPath], structs...)
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Failed to walk directory tree: %v", err)
	}

	// Генерируем файлы reset.gen.go для каждого пакета
	for pkgPath, structs := range structsByPackage {
		if err := generateResetFile(pkgPath, structs); err != nil {
			log.Printf("Failed to generate reset file for %s: %v", pkgPath, err)
		}
	}

	fmt.Printf("Generated Reset() methods for %d package(s)\n", len(structsByPackage))
}

// findResetableStructs ищет структуры с комментарием // generate:reset
func findResetableStructs(file *ast.File, filePath string) []StructInfo {
	var structs []StructInfo

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Проверяем комментарий
			if !hasResetComment(genDecl.Doc) && !hasResetComment(typeSpec.Doc) {
				continue
			}

			// Собираем информацию о полях
			fields := extractFields(structType)

			structs = append(structs, StructInfo{
				Name:    typeSpec.Name.Name,
				Fields:  fields,
				PkgName: file.Name.Name,
				PkgPath: filepath.Dir(filePath),
			})
		}
	}

	return structs
}

// hasResetComment проверяет наличие комментария // generate:reset
func hasResetComment(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, comment := range doc.List {
		if strings.Contains(comment.Text, "generate:reset") {
			return true
		}
	}
	return false
}

// extractFields извлекает информацию о полях структуры
func extractFields(structType *ast.StructType) []FieldInfo {
	var fields []FieldInfo

	for _, field := range structType.Fields.List {
		// Пропускаем встроенные поля без имен
		if len(field.Names) == 0 {
			continue
		}

		for _, name := range field.Names {
			// Пропускаем неэкспортированные поля (начинаются с маленькой буквы)
			// На самом деле, нужно обрабатывать все поля, включая неэкспортированные
			fieldInfo := FieldInfo{
				Name: name.Name,
			}

			// Определяем тип поля
			fieldInfo.Type = exprToString(field.Type)
			fieldInfo.IsPointer = isPointerType(field.Type)
			fieldInfo.IsSlice = isSliceType(field.Type)
			fieldInfo.IsMap = isMapType(field.Type)
			fieldInfo.IsStruct = isStructType(field.Type)

			fields = append(fields, fieldInfo)
		}
	}

	return fields
}

// exprToString преобразует ast.Expr в строку типа
func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// isPointerType проверяет, является ли тип указателем
func isPointerType(expr ast.Expr) bool {
	_, ok := expr.(*ast.StarExpr)
	return ok
}

// isSliceType проверяет, является ли тип слайсом
func isSliceType(expr ast.Expr) bool {
	arrayType, ok := expr.(*ast.ArrayType)
	return ok && arrayType.Len == nil
}

// isMapType проверяет, является ли тип мапой
func isMapType(expr ast.Expr) bool {
	_, ok := expr.(*ast.MapType)
	return ok
}

// isStructType проверяет, является ли тип структурой
func isStructType(expr ast.Expr) bool {
	// Убираем указатель если есть
	if starExpr, ok := expr.(*ast.StarExpr); ok {
		expr = starExpr.X
	}

	// Проверяем, является ли это пользовательским типом (вероятно структурой)
	_, isIdent := expr.(*ast.Ident)
	_, isSelector := expr.(*ast.SelectorExpr)
	return isIdent || isSelector
}

// generateResetFile генерирует файл reset.gen.go для пакета
func generateResetFile(pkgPath string, structs []StructInfo) error {
	var buf bytes.Buffer

	// Заголовок
	buf.WriteString("// Code generated by cmd/reset; DO NOT EDIT.\n\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", structs[0].PkgName))

	// Генерируем методы Reset() для каждой структуры
	for _, s := range structs {
		buf.WriteString(generateResetMethod(s))
		buf.WriteString("\n")
	}

	// Форматируем код
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format generated code: %w", err)
	}

	// Записываем в файл
	outputPath := filepath.Join(pkgPath, "reset.gen.go")
	if err := os.WriteFile(outputPath, formatted, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Generated %s\n", outputPath)
	return nil
}

// generateResetMethod генерирует метод Reset() для структуры
func generateResetMethod(s StructInfo) string {
	var buf bytes.Buffer

	// Сигнатура метода
	buf.WriteString(fmt.Sprintf("func (r *%s) Reset() {\n", s.Name))
	buf.WriteString("\tif r == nil {\n")
	buf.WriteString("\t\treturn\n")
	buf.WriteString("\t}\n\n")

	// Генерируем код для каждого поля
	for _, field := range s.Fields {
		buf.WriteString(generateFieldReset(field))
	}

	buf.WriteString("}\n")

	return buf.String()
}

// generateFieldReset генерирует код сброса для поля
func generateFieldReset(field FieldInfo) string {
	fieldName := "r." + field.Name

	if field.IsPointer {
		// Указатель
		return generatePointerReset(fieldName, field)
	} else if field.IsSlice {
		// Слайс
		return fmt.Sprintf("\t%s = %s[:0]\n", fieldName, fieldName)
	} else if field.IsMap {
		// Мапа
		return fmt.Sprintf("\tclear(%s)\n", fieldName)
	} else if isPrimitiveType(field.Type) {
		// Примитивный тип
		return fmt.Sprintf("\t%s = %s\n", fieldName, getZeroValue(field.Type))
	} else if field.IsStruct {
		// Вложенная структура - пытаемся вызвать Reset()
		return generateStructReset(fieldName, field)
	} else {
		// Для всех остальных случаев (интерфейсы, каналы, функции) - nil
		return fmt.Sprintf("\t%s = nil\n", fieldName)
	}
}

// generatePointerReset генерирует код сброса для указателя
func generatePointerReset(fieldName string, field FieldInfo) string {
	var buf bytes.Buffer

	// Убираем * из типа для получения базового типа
	baseType := strings.TrimPrefix(field.Type, "*")

	buf.WriteString(fmt.Sprintf("\tif %s != nil {\n", fieldName))

	// Определяем, что за тип под указателем
	if strings.HasPrefix(baseType, "[]") {
		// Слайс под указателем
		buf.WriteString(fmt.Sprintf("\t\t*%s = (*%s)[:0]\n", fieldName, fieldName))
	} else if strings.HasPrefix(baseType, "map[") {
		// Мапа под указателем
		buf.WriteString(fmt.Sprintf("\t\tclear(*%s)\n", fieldName))
	} else if isPrimitiveType(baseType) {
		// Примитив под указателем
		buf.WriteString(fmt.Sprintf("\t\t*%s = %s\n", fieldName, getZeroValue(baseType)))
	} else {
		// Вероятно структура - пытаемся вызвать Reset()
		buf.WriteString(fmt.Sprintf("\t\tif resetter, ok := interface{}(%s).(interface{ Reset() }); ok {\n", fieldName))
		buf.WriteString("\t\t\tresetter.Reset()\n")
		buf.WriteString("\t\t}\n")
	}

	buf.WriteString("\t}\n")

	return buf.String()
}

// generateStructReset генерирует код сброса для вложенной структуры
func generateStructReset(fieldName string, field FieldInfo) string {
	var buf bytes.Buffer

	// Пытаемся вызвать Reset() если метод существует
	buf.WriteString(fmt.Sprintf("\tif resetter, ok := interface{}(&%s).(interface{ Reset() }); ok {\n", fieldName))
	buf.WriteString("\t\tresetter.Reset()\n")
	buf.WriteString("\t}\n")

	return buf.String()
}

// isPrimitiveType проверяет, является ли тип примитивным
func isPrimitiveType(t string) bool {
	primitives := []string{
		"bool",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"complex64", "complex128",
		"string",
		"byte", "rune",
	}

	for _, p := range primitives {
		if t == p {
			return true
		}
	}
	return false
}

// getZeroValue возвращает нулевое значение для типа
func getZeroValue(t string) string {
	switch t {
	case "bool":
		return "false"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"complex64", "complex128",
		"byte", "rune":
		return "0"
	case "string":
		return `""`
	default:
		// Для всех остальных типов (интерфейсы, каналы, функции)
		return "nil"
	}
}
