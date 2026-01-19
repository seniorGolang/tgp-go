// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package model

// GetAnnotationValue возвращает значение аннотации с учетом наследования сверху вниз:
// проект → контракт → метод → переменная/поле
//
// Параметры:
//   - project: проект (может быть nil)
//   - contract: контракт (может быть nil)
//   - method: метод (может быть nil)
//   - variable: переменная/поле (может быть nil)
//   - tagName: имя аннотации
//   - defaultValue: значение по умолчанию (если аннотация не найдена)
//
// Возвращает значение аннотации или значение по умолчанию.
func GetAnnotationValue(project *Project, contract *Contract, method *Method, variable *Variable, tagName string, defaultValue ...string) (value string) {

	// Проверяем переменную/поле (самый низкий уровень)
	if variable != nil && variable.Annotations != nil {
		if val, found := variable.Annotations[tagName]; found && val != "" {
			return val
		}
	}

	// Проверяем метод
	if method != nil && method.Annotations != nil {
		if val, found := method.Annotations[tagName]; found && val != "" {
			return val
		}
	}

	// Проверяем контракт
	if contract != nil && contract.Annotations != nil {
		if val, found := contract.Annotations[tagName]; found && val != "" {
			return val
		}
	}

	// Проверяем проект (самый верхний уровень)
	if project != nil && project.Annotations != nil {
		return project.Annotations.Value(tagName, defaultValue...)
	}

	// Если ничего не найдено, возвращаем значение по умолчанию
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// GetAnnotationValueInt возвращает целочисленное значение аннотации с учетом наследования.
func GetAnnotationValueInt(project *Project, contract *Contract, method *Method, variable *Variable, tagName string, defaultValue ...int) (value int) {

	// Проверяем переменную/поле (самый низкий уровень)
	if variable != nil && variable.Annotations != nil && variable.Annotations.IsSet(tagName) {
		return variable.Annotations.ValueInt(tagName, defaultValue...)
	}

	// Проверяем метод
	if method != nil && method.Annotations != nil && method.Annotations.IsSet(tagName) {
		return method.Annotations.ValueInt(tagName, defaultValue...)
	}

	// Проверяем контракт
	if contract != nil && contract.Annotations != nil && contract.Annotations.IsSet(tagName) {
		return contract.Annotations.ValueInt(tagName, defaultValue...)
	}

	// Проверяем проект (самый верхний уровень)
	if project != nil && project.Annotations != nil {
		return project.Annotations.ValueInt(tagName, defaultValue...)
	}

	// Если ничего не найдено, возвращаем значение по умолчанию
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

// GetAnnotationValueBool возвращает булево значение аннотации с учетом наследования.
func GetAnnotationValueBool(project *Project, contract *Contract, method *Method, variable *Variable, tagName string, defaultValue ...bool) (value bool) {

	// Проверяем переменную/поле (самый низкий уровень)
	if variable != nil && variable.Annotations != nil && variable.Annotations.IsSet(tagName) {
		return variable.Annotations.ValueBool(tagName, defaultValue...)
	}

	// Проверяем метод
	if method != nil && method.Annotations != nil && method.Annotations.IsSet(tagName) {
		return method.Annotations.ValueBool(tagName, defaultValue...)
	}

	// Проверяем контракт
	if contract != nil && contract.Annotations != nil && contract.Annotations.IsSet(tagName) {
		return contract.Annotations.ValueBool(tagName, defaultValue...)
	}

	// Проверяем проект (самый верхний уровень)
	if project != nil && project.Annotations != nil {
		return project.Annotations.ValueBool(tagName, defaultValue...)
	}

	// Если ничего не найдено, возвращаем значение по умолчанию
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return false
}

// IsAnnotationSet проверяет, установлена ли аннотация на любом уровне иерархии.
func IsAnnotationSet(project *Project, contract *Contract, method *Method, variable *Variable, tagName string) (found bool) {

	// Проверяем переменную/поле (самый низкий уровень)
	if variable != nil && variable.Annotations != nil {
		if variable.Annotations.IsSet(tagName) {
			return true
		}
	}

	// Проверяем метод
	if method != nil && method.Annotations != nil {
		if method.Annotations.IsSet(tagName) {
			return true
		}
	}

	// Проверяем контракт
	if contract != nil && contract.Annotations != nil {
		if contract.Annotations.IsSet(tagName) {
			return true
		}
	}

	// Проверяем проект (самый верхний уровень)
	if project != nil && project.Annotations != nil {
		return project.Annotations.IsSet(tagName)
	}

	return false
}
