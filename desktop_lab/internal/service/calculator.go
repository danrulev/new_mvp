package service

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

type tokenType int

const (
	tokNumber tokenType = iota
	tokIdentifier
	tokOperator
	tokLeftParen
	tokRightParen
	tokComma
	tokFunctionArgCount
)

type token struct {
	typ      tokenType
	value    string
	argCount int
}

// Приоритеты операторов
var operators = map[string]struct {
	precedence int
	rightAssoc bool
}{
	"+": {1, false},
	"-": {1, false},
	"*": {2, false},
	"/": {2, false},
	"^": {3, true},
}

var builtInFunctions = map[string]func(args ...float64) (float64, error){
	// Тригонометрия
	"sin": func(a ...float64) (float64, error) {
		if len(a) != 1 {
			return 0, fmt.Errorf("sin требует 1 аргумент, получено %d", len(a))
		}
		return math.Sin(a[0]), nil
	},
	"cos": func(a ...float64) (float64, error) {
		if len(a) != 1 {
			return 0, fmt.Errorf("cos требует 1 аргумент, получено %d", len(a))
		}
		return math.Cos(a[0]), nil
	},
	"tan": func(a ...float64) (float64, error) {
		if len(a) != 1 {
			return 0, fmt.Errorf("tan требует 1 аргумент, получено %d", len(a))
		}
		return math.Tan(a[0]), nil
	},
	"asin": func(a ...float64) (float64, error) {
		if len(a) != 1 {
			return 0, fmt.Errorf("asin требует 1 аргумент")
		}
		return math.Asin(a[0]), nil
	},
	"acos": func(a ...float64) (float64, error) {
		if len(a) != 1 {
			return 0, fmt.Errorf("acos требует 1 аргумент")
		}
		return math.Acos(a[0]), nil
	},
	"atan": func(a ...float64) (float64, error) {
		if len(a) != 1 {
			return 0, fmt.Errorf("atan требует 1 аргумент")
		}
		return math.Atan(a[0]), nil
	},

	// Корни и степени
	"sqrt": func(a ...float64) (float64, error) {
		if len(a) != 1 {
			return 0, fmt.Errorf("sqrt требует 1 аргумент")
		}
		if a[0] < 0 {
			return 0, errors.New("квадратный корень из отрицательного числа")
		}
		return math.Sqrt(a[0]), nil
	},
	"pow": func(a ...float64) (float64, error) {
		if len(a) != 2 {
			return 0, fmt.Errorf("pow требует 2 аргумента (base, exp)")
		}
		return math.Pow(a[0], a[1]), nil
	},
	"exp": func(a ...float64) (float64, error) {
		if len(a) != 1 {
			return 0, fmt.Errorf("exp требует 1 аргумент")
		}
		return math.Exp(a[0]), nil
	},

	// Логарифмы
	"log": func(a ...float64) (float64, error) {
		if len(a) != 1 {
			return 0, fmt.Errorf("log (натуральный) требует 1 аргумент")
		}
		if a[0] <= 0 {
			return 0, errors.New("логарифм от неположительного числа")
		}
		return math.Log(a[0]), nil
	},
	"log10": func(a ...float64) (float64, error) {
		if len(a) != 1 {
			return 0, fmt.Errorf("log10 требует 1 аргумент")
		}
		if a[0] <= 0 {
			return 0, errors.New("логарифм от неположительного числа")
		}
		return math.Log10(a[0]), nil
	},
	"log2": func(a ...float64) (float64, error) {
		if len(a) != 1 {
			return 0, fmt.Errorf("log2 требует 1 аргумент")
		}
		if a[0] <= 0 {
			return 0, errors.New("логарифм от неположительного числа")
		}
		return math.Log2(a[0]), nil
	},

	// Модуль и округление
	"abs": func(a ...float64) (float64, error) {
		if len(a) != 1 {
			return 0, fmt.Errorf("abs требует 1 аргумент")
		}
		return math.Abs(a[0]), nil
	},
	"floor": func(a ...float64) (float64, error) {
		if len(a) != 1 {
			return 0, fmt.Errorf("floor требует 1 аргумент")
		}
		return math.Floor(a[0]), nil
	},
	"ceil": func(a ...float64) (float64, error) {
		if len(a) != 1 {
			return 0, fmt.Errorf("ceil требует 1 аргумент")
		}
		return math.Ceil(a[0]), nil
	},
	"round": func(a ...float64) (float64, error) {
		if len(a) != 1 {
			return 0, fmt.Errorf("round требует 1 аргумент")
		}
		return math.Round(a[0]), nil
	},

	// Агрегатные функции (переменное число аргументов)
	"min": func(a ...float64) (float64, error) {
		if len(a) == 0 {
			return 0, errors.New("min требует хотя бы 1 аргумент")
		}
		m := a[0]
		for _, v := range a[1:] {
			if v < m {
				m = v
			}
		}
		return m, nil
	},
	"max": func(a ...float64) (float64, error) {
		if len(a) == 0 {
			return 0, errors.New("max требует хотя бы 1 аргумент")
		}
		m := a[0]
		for _, v := range a[1:] {
			if v > m {
				m = v
			}
		}
		return m, nil
	},
	"sum": func(a ...float64) (float64, error) {
		var s float64
		for _, v := range a {
			s += v
		}
		return s, nil
	},
	"avg": func(a ...float64) (float64, error) {
		if len(a) == 0 {
			return 0, errors.New("avg требует хотя бы 1 аргумент")
		}
		var s float64
		for _, v := range a {
			s += v
		}
		return s / float64(len(a)), nil
	},
}

// Константы (обрабатываются как функции без аргументов или как идентификаторы)
var constants = map[string]float64{
	"pi": math.Pi,
	"π":  math.Pi,
	"e":  math.E,
	"φ":  math.Phi,
}

// CalculateFormula - публичный метод для вычисления формул (используется из репозитория)
func (s *ProtocolService) CalculateFormula(exprStr string, params map[string]interface{}) (float64, error) {
	if strings.TrimSpace(exprStr) == "" {
		return 0, errors.New("пустая формула")
	}

	// 1. Токенизация
	tokens, err := tokenize(exprStr)
	if err != nil {
		return 0, err
	}

	// 2. Преобразование в Обратную Польскую Запись (RPN)
	rpn, err := toRPN(tokens)
	if err != nil {
		return 0, err
	}

	// 3. Вычисление
	return evalRPN(rpn, params)
}

// calculateFormula - приватная версия для внутреннего использования
func (s *ProtocolService) calculateFormula(exprStr string, params map[string]interface{}) (float64, error) {
	return s.CalculateFormula(exprStr, params)
}

func tokenize(expr string) ([]token, error) {
	var tokens []token
	i := 0
	n := len(expr)

	for i < n {
		ch := expr[i]

		if unicode.IsSpace(rune(ch)) {
			i++
			continue
		}

		// Число
		if unicode.IsDigit(rune(ch)) || (ch == '.' && i+1 < n && unicode.IsDigit(rune(expr[i+1]))) {
			start := i
			hasDot := ch == '.'
			hasExp := false

			for i < n {
				c := expr[i]
				if unicode.IsDigit(rune(c)) {
					i++
				} else if c == '.' && !hasDot && !hasExp {
					hasDot = true
					i++
				} else if (c == 'e' || c == 'E') && !hasExp && i > start {
					if i+1 < n && (expr[i+1] == '+' || expr[i+1] == '-') {
						i += 2
					} else if i+1 < n && unicode.IsDigit(rune(expr[i+1])) {
						i++
					} else {
						break
					}
					hasExp = true
				} else {
					break
				}
			}
			tokens = append(tokens, token{typ: tokNumber, value: expr[start:i]})
			continue
		}

		// Идентификатор
		if unicode.IsLetter(rune(ch)) || ch == '_' {
			start := i
			for i < n && (unicode.IsLetter(rune(expr[i])) || unicode.IsDigit(rune(expr[i])) || expr[i] == '_') {
				i++
			}
			val := expr[start:i]
			lowerVal := strings.ToLower(val)

			// Константы - регистронезависимые
			if constVal, ok := constants[lowerVal]; ok {
				tokens = append(tokens, token{typ: tokNumber, value: fmt.Sprintf("%.15f", constVal)})
				continue
			}

			// Сохраняем оригинальный регистр для переменных и функций
			tokens = append(tokens, token{typ: tokIdentifier, value: val})
			continue
		}

		// Операторы
		switch ch {
		case '+', '*', '/', '^':
			tokens = append(tokens, token{typ: tokOperator, value: string(ch)})
			i++
		case '-':
			// Унарный минус - создаём специальный токен
			isUnary := false
			if len(tokens) == 0 {
				isUnary = true
			} else {
				prev := tokens[len(tokens)-1]
				if prev.typ == tokOperator || prev.typ == tokLeftParen || prev.typ == tokComma {
					isUnary = true
				}
			}

			if isUnary {
				tokens = append(tokens, token{typ: tokOperator, value: "u-"})
			} else {
				tokens = append(tokens, token{typ: tokOperator, value: "-"})
			}
			i++

		case '(':
			tokens = append(tokens, token{typ: tokLeftParen, value: "("})
			i++
		case ')':
			tokens = append(tokens, token{typ: tokRightParen, value: ")"})
			i++
		case ',':
			tokens = append(tokens, token{typ: tokComma, value: ","})
			i++
		default:
			return nil, fmt.Errorf("недопустимый символ '%c' в позиции %d", ch, i)
		}
	}

	return tokens, nil
}

func toRPN(tokens []token) ([]token, error) {
	var output []token
	var opStack []token
	var funcArgStack []int

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]

		switch t.typ {
		case tokNumber:
			output = append(output, t)

		case tokIdentifier:
			isFunction := false
			// Проверяем встроенные функции регистронезависимо
			if _, exists := builtInFunctions[strings.ToLower(t.value)]; exists {
				if i+1 < len(tokens) && tokens[i+1].typ == tokLeftParen {
					isFunction = true
				}
			}

			if isFunction {
				opStack = append(opStack, token{
					typ:      tokIdentifier,
					value:    strings.ToLower(t.value), // Сохраняем в нижнем регистре для поиска в builtInFunctions
					argCount: 0,
				})
				funcArgStack = append(funcArgStack, 0)
			} else {
				output = append(output, t) // Сохраняем оригинальный регистр для переменных
			}

		case tokOperator:
			o1 := t.value
			o1Info, ok1 := operators[o1]
			if !ok1 {
				return nil, fmt.Errorf("неизвестный оператор '%s'", o1)
			}

			for len(opStack) > 0 {
				top := opStack[len(opStack)-1]
				if top.typ != tokOperator {
					break
				}
				o2 := top.value
				o2Info, ok2 := operators[o2]
				if !ok2 {
					break
				}

				// Обработка унарного минуса (u-)
				if o1 == "u-" {
					break
				}
				if o2 == "u-" {
					output = append(output, opStack[len(opStack)-1])
					opStack = opStack[:len(opStack)-1]
					continue
				}

				if (!o1Info.rightAssoc && o1Info.precedence <= o2Info.precedence) ||
					(o1Info.rightAssoc && o1Info.precedence < o2Info.precedence) {
					output = append(output, opStack[len(opStack)-1])
					opStack = opStack[:len(opStack)-1]
				} else {
					break
				}
			}
			opStack = append(opStack, t)

		case tokLeftParen:
			opStack = append(opStack, t)

		case tokComma:
			if len(funcArgStack) == 0 {
				return nil, errors.New("запятая вне вызова функции")
			}
			funcArgStack[len(funcArgStack)-1]++

			for len(opStack) > 0 && opStack[len(opStack)-1].typ != tokLeftParen {
				output = append(output, opStack[len(opStack)-1])
				opStack = opStack[:len(opStack)-1]
			}
			if len(opStack) == 0 {
				return nil, errors.New("ошибка синтаксиса: лишняя запятая")
			}

		case tokRightParen:
			foundLeft := false
			for len(opStack) > 0 {
				top := opStack[len(opStack)-1]
				opStack = opStack[:len(opStack)-1]
				if top.typ == tokLeftParen {
					foundLeft = true
					break
				}
				output = append(output, top)
			}
			if !foundLeft {
				return nil, errors.New("ошибка синтаксиса: лишняя закрывающая скобка")
			}

			// Увеличиваем счётчик аргументов (последний аргумент завершён)
			if len(funcArgStack) > 0 {
				funcArgStack[len(funcArgStack)-1]++
			}

			// Выталкиваем функцию
			if len(opStack) > 0 && opStack[len(opStack)-1].typ == tokIdentifier {
				if _, isFunc := builtInFunctions[opStack[len(opStack)-1].value]; isFunc {
					fnToken := opStack[len(opStack)-1]
					opStack = opStack[:len(opStack)-1]

					args := 0
					if len(funcArgStack) > 0 {
						args = funcArgStack[len(funcArgStack)-1]
						funcArgStack = funcArgStack[:len(funcArgStack)-1]
					}

					fnToken.typ = tokFunctionArgCount
					fnToken.argCount = args
					output = append(output, fnToken)
				}
			}
		}
	}

	for len(opStack) > 0 {
		top := opStack[len(opStack)-1]
		opStack = opStack[:len(opStack)-1]
		if top.typ == tokLeftParen || top.typ == tokRightParen {
			return nil, errors.New("ошибка синтаксиса: несоответствие скобок")
		}
		output = append(output, top)
	}

	if len(funcArgStack) > 0 {
		return nil, errors.New("ошибка синтаксиса: функция не закрыта скобками")
	}

	return output, nil
}

func evalRPN(rpn []token, params map[string]interface{}) (float64, error) {
	var stack []float64

	for _, t := range rpn {
		switch t.typ {
		case tokNumber:
			val, err := strconv.ParseFloat(t.value, 64)
			if err != nil {
				return 0, fmt.Errorf("ошибка парсинга числа '%s': %w", t.value, err)
			}
			stack = append(stack, val)

		case tokIdentifier:
			// Ищем переменную с учетом регистра
			rawVal, exists := params[t.value]
			if !exists {
				// Может быть константа?
				if constVal, ok := constants[strings.ToLower(t.value)]; ok {
					stack = append(stack, constVal)
					continue
				}
				return 0, fmt.Errorf("переменная '%s' не найдена", t.value)
			}

			var fVal float64
			switch v := rawVal.(type) {
			case float64:
				fVal = v
			case float32:
				fVal = float64(v)
			case int:
				fVal = float64(v)
			case int64:
				fVal = float64(v)
			case int32:
				fVal = float64(v)
			default:
				return 0, fmt.Errorf("неподдерживаемый тип переменной '%s': %T", t.value, v)
			}
			stack = append(stack, fVal)

		case tokFunctionArgCount:
			fnName := t.value
			fn, exists := builtInFunctions[fnName]
			if !exists {
				return 0, fmt.Errorf("неизвестная функция '%s'", fnName)
			}

			argCount := t.argCount
			if len(stack) < argCount {
				return 0, fmt.Errorf("функция '%s' требует %d аргументов, в стеке %d", fnName, argCount, len(stack))
			}

			args := make([]float64, argCount)
			for i := argCount - 1; i >= 0; i-- {
				args[i] = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			}

			res, err := fn(args...)
			if err != nil {
				return 0, fmt.Errorf("ошибка функции '%s': %w", fnName, err)
			}
			stack = append(stack, res)

		case tokOperator:
			// Унарный оператор
			if t.value == "u-" {
				if len(stack) < 1 {
					return 0, errors.New("недостаточно операндов для унарного минуса")
				}
				a := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				stack = append(stack, -a)
				continue
			}

			// Бинарный оператор
			if len(stack) < 2 {
				return 0, errors.New("недостаточно операндов для оператора")
			}
			b := stack[len(stack)-1]
			a := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			var res float64
			switch t.value {
			case "+":
				res = a + b
			case "-":
				res = a - b
			case "*":
				res = a * b
			case "/":
				if b == 0 {
					return 0, errors.New("деление на ноль")
				}
				res = a / b
			case "^":
				res = math.Pow(a, b)
			default:
				return 0, fmt.Errorf("неизвестный оператор '%s'", t.value)
			}
			stack = append(stack, res)
		}
	}

	if len(stack) != 1 {
		return 0, fmt.Errorf("ошибка: в стеке %d значений вместо 1", len(stack))
	}

	return stack[0], nil
}
