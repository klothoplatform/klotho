import Multimap = require('multimap')
import * as sha256 from 'simple-sha256'

export interface SanitizationOptions {
    maxLength?: number
    minLength?: number
    rules: Array<SanitizationRule>
    maxPasses?: number
}

interface ResourceName {
    prefix?: string
    name?: string
    suffix?: string
    separator?: string
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

export function sanitizeResourceName(
    s: string,
    options: Partial<SanitizationOptions>
): SanitizationResult {
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

export function validateResourceName(name: string, options: Partial<SanitizationOptions>) {
    const violations = doValidateResourceName(name, options)
    if (violations.length > 0) {
        throw new Error(`Invalid resource name '${name}':\n\t${violations.join('\n\t')}`)
    }
}

function doValidateResourceName(
    name: string,
    options: Partial<SanitizationOptions>
): Array<string> {
    const violations = new Array<string>()
    if (options.minLength != null && name.length < options.minLength) {
        violations.push(
            `Invalid resource name: "${name}": length < minLength (${options.minLength})`
        )
    }
    if (options.maxLength != null && name.length > options.maxLength) {
        violations.push(
            `Invalid resource name: "${name}": length > maxLength (${options.maxLength})`
        )
    }

    violations.push(
        ...(options.rules
            ?.filter((r) => !r.validate(name))
            .map((r) => `Naming rule violated: ${r.description}`) || [])
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
        validate: (resourceName) => pattern.test(resourceName),
        fix,
    }
}

export function regexpNotMatch(
    name: string,
    pattern: RegExp,
    fix: FixFunc | undefined = undefined
): SanitizationRule {
    return {
        description: name,
        validate: (resourceName) => !pattern.test(resourceName),
        fix,
    }
}

type ShorteningStrategy = (name: string, maxLength: number) => string

export interface Component {
    content: string
    priority?: number
    shorteningStrategy?: ShorteningStrategy
}

function shortenName(
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

export function resourceName(options: SanitizationOptions) {
    return function (
        strings: TemplateStringsArray,
        ...components: Array<string | Component>
    ): string {
        const shortenedName = shortenName(strings, components, options)
        const { result: sanitizedName, violations } = sanitizeResourceName(shortenedName, options)
        if (violations.length > 0) {
            throw new Error(`Resource name generation failed:\n\t${violations.join('\n\t')}`)
        }
        return sanitizedName
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
