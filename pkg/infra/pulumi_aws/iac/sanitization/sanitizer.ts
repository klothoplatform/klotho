import Multimap = require('multimap')
import * as sha256 from 'simple-sha256'

export interface SanitizationOptions {
    maxLength?: number
    minLength?: number
    rules: Array<SanitizationRule>
    maxPasses?: number
}

export interface SanitizationResult {
    result: string
    violations: Array<string>
}

export interface SanitizationRule extends ValidationRule {
    fix?: FixFunc
}

type FixFunc = (string) => string

export interface ValidationRule {
    description: string

    validate(string): boolean
}

export function sanitize(s: string, options: Partial<SanitizationOptions>): SanitizationResult {
    let result = s
    let failedRules = new Array<SanitizationRule>()
    for (let i = 0; i < (options.maxPasses || 5); i++) {
        let failedRules = options.rules?.filter((r) => !r.validate(result))
        failedRules?.forEach((f) => console.debug(f))
        if (options.minLength != null && result.length < options.minLength) {
            throw new Error(
                `The sanitized value, "${result}", is shorten than minLength: ${options.minLength}`
            )
        }
        if (options.maxLength != null && result.length > options.maxLength) {
            result = result.substring(0, options.maxLength)
        }
        if (failedRules?.length === 0) {
            return { result: result, violations: [] }
        }
        failedRules?.forEach((r) => (result = r.fix?.apply(this, [result]) || result))
        failedRules = options.rules?.filter((r) => !r.validate(result))
        failedRules?.forEach((f) => console.debug(f))
        if (failedRules?.length === 0) {
            return { result, violations: [] }
        }
    }
    return { result, violations: failedRules.map((r) => r.description) }
}

export function validate(input: string, options: Partial<SanitizationOptions>) {
    const violations = doValidate(input, options)
    if (violations.length > 0) {
        throw new Error(`Invalid input '${input}':\n\t${violations.join('\n\t')}`)
    }
}

function doValidate(input: string, options: Partial<SanitizationOptions>): Array<string> {
    const violations = new Array<string>()
    if (options.minLength != null && input.length < options.minLength) {
        violations.push(`Invalid input: "${input}": length < minLength (${options.minLength})`)
    }
    if (options.maxLength != null && input.length > options.maxLength) {
        violations.push(`Invalid input: "${input}": length > maxLength (${options.maxLength})`)
    }

    violations.push(
        ...(options.rules
            ?.filter((r) => !r.validate(input))
            .map((r) => `validation rule violated: ${r.description}`) || [])
    )
    return violations
}

function hashComponent(input: string, maxLength: number): string {
    maxLength = maxLength > 5 ? 5 : maxLength
    const hash = sha256.sync(input)
    return hash.substring(0, maxLength <= input.length ? maxLength : input.length)
}

export function regexpMatch(
    description: string,
    pattern: RegExp,
    fix: FixFunc | undefined = undefined
): SanitizationRule {
    return {
        description: description
            ? description
            : `The supplied string must match the following pattern: ${pattern.source}`,
        validate: (input) => pattern.test(input),
        fix,
    }
}

export function regexpNotMatch(
    input: string,
    pattern: RegExp,
    fix: FixFunc | undefined = undefined
): SanitizationRule {
    return {
        description: input,
        validate: (input) => !pattern.test(input),
        fix,
    }
}

type ShorteningStrategy = (input: string, maxLength: number) => string

export interface Component {
    content: string
    priority?: number
    shorteningStrategy?: ShorteningStrategy
}

function shortenString(
    strings: TemplateStringsArray,
    components: Array<string | Component>,
    options: SanitizationOptions
) {
    const resolvedComponents: Array<Component> = components.map((c) => {
        return typeof c === 'string' ? { content: c } : { ...c } // clone each component to avoid unintended modification to inputs
    })
    let length =
        arrTextLen([...strings.raw], (s) => s.length) +
        arrTextLen(resolvedComponents, (a) => a.content.length)

    let componentsByPriority = new Multimap<number, Component>()
    resolvedComponents.forEach((c) =>
        componentsByPriority.set(c.priority === undefined ? 100 : c.priority, c)
    )
    const sorted = [...componentsByPriority.keys()].sort()

    if (options.maxLength != null) {
        for (const p of sorted) {
            if (length <= options.maxLength) {
                break
            }
            for (const c of componentsByPriority.get(p)) {
                if (length <= options.maxLength) {
                    break
                }
                const oldCLen = c.content.length
                c.content = c.shorteningStrategy
                    ? c.shorteningStrategy(
                          c.content,
                          options.maxLength - (length - c.content.length)
                      )
                    : c.content
                length -= oldCLen - c.content.length
            }
        }
    }

    let result = ''
    for (let i = 0; i < strings.length; i++) {
        result += (strings[i] || '') + (resolvedComponents[i]?.content || '')
    }
    return result
}

export function sanitized(options: SanitizationOptions) {
    return function (
        strings: TemplateStringsArray,
        ...components: Array<string | Component>
    ): string {
        const shortenedString = shortenString(strings, components, options)
        const { result, violations } = sanitize(shortenedString, options)
        if (violations.length > 0) {
            throw new Error(`sanitization failed:\n\t${violations.join('\n\t')}`)
        }
        return result
    }
}

function arrTextLen<T>(arr: Array<T>, lenFunc: (arg0: T) => number): number {
    return arr.reduce((p, c, i) => p + lenFunc(c), 0)
}

export function h(content: string, priority: number | undefined = undefined): Component {
    return {
        content,
        priority,
        shorteningStrategy: hashComponent,
    }
}

export function t(content: string, priority: number | undefined = undefined): Component {
    return {
        content,
        priority,
        shorteningStrategy: (t, m) => t.substring(0, m),
    }
}
