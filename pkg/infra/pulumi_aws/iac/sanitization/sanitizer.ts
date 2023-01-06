import {json} from "stream/consumers";
import * as crypto from 'crypto'

export interface SanitizationOptions {
    maxLength: number
    minLength: number
    rules: Array<SanitizationRule>
    maxPasses?: number
}


interface ResourceName {
    prefix?: string
    name?: string
    suffix?: string
    separator?: string
}


export function generateValidResourceName(name: ResourceName | string, options: ResourceNameOptions): string {
    if (typeof name === "string") {
        name = {name}
    }

    const shortenedName = options.maxLength && options.shorteningStrategy ? options.shorteningStrategy(name, options.maxLength) : name;
    const resolvedName = options.namingStrategy?.apply(this, [shortenedName]) || simpleMerge(shortenedName);

    const {result: sanitizedName, violations} = sanitizeResourceName(resolvedName, options)
    if (violations.length > 0) {
        throw new Error(`Resource name generation failed:\n\t${violations.join("\n\t")}`);
    }
    return sanitizedName;
}

export interface SanitizationResult {
    result: string
    violations: Array<string>
}

export function sanitizeResourceName(s: string, options: Partial<SanitizationOptions>): SanitizationResult {
    let result = s;
    let failedRules = new Array<SanitizationRule>();
    for (let i = 0; i < (options.maxPasses || 5); i++) {
        let failedRules = options.rules?.filter(r => !r.validate(result));
        failedRules?.forEach(f => console.debug(f))
        if (options.minLength != null && result.length < options.minLength) {
            throw new Error(`The sanitized value, "${result}", is shorten than minLength: ${options.minLength}`);
        }
        if (options.maxLength != null && result.length > options.maxLength) {
            result = result.substring(0, options.maxLength);
        }
        if (failedRules?.length === 0) {
            return {result: result, violations: []};
        }
        failedRules?.forEach(r => result = r.fix?.apply(this, [result]) || result);
        failedRules = options.rules?.filter(r => !r.validate(result));
        failedRules?.forEach(f => console.debug(f))
        if (failedRules?.length === 0) {
            return {result, violations: []};
        }
    }
    return {result, violations: failedRules.map(r => r.description)};
}

export function validateResourceName(name: string, options: Partial<SanitizationOptions>): Array<string> {
    const violations = new Array<string>();
    if (options.minLength != null && name.length < options.minLength) {
        violations.push(`Invalid resource name: "${name}": length < minLength (${options.minLength})`)
    }
    if (options.maxLength != null && name.length > options.maxLength) {
        violations.push(`Invalid resource name: "${name}": length > maxLength (${options.maxLength})`)
    }

    violations.push(...(options.rules?.filter(r => !r.validate(name)).map(r => `Naming rule violated: ${r.description}`) || []));
    return violations;
}

function simpleMerge(resourceName: ResourceName): string {
    const prefix = resourceName.prefix ? resourceName.prefix : "";
    const name = resourceName.name ? resourceName.name : "";
    const suffix = resourceName.suffix ? resourceName.suffix : "";
    const separator = resourceName.separator ? resourceName.separator : "";
    return `${prefix}${prefix ? separator : ""}${name}${suffix ? separator : ""}${suffix}`
}

export interface SanitizationRule extends ValidationRule {
    fix?: FixFunc
}

type FixFunc = (string) => string

export interface ValidationRule {
    description: string

    validate(string): boolean
}

type NamingStrategy = (resourceName: ResourceName) => string
type ShorteningStrategy = (name: ResourceName, maxLength: number) => ResourceName

interface ResourceNameOptions extends Partial<SanitizationOptions> {
    namingStrategy?: NamingStrategy
    shorteningStrategy?: ShorteningStrategy
}


export function truncatePrefix(resourceName: ResourceName, maxLength: number): ResourceName {
    const prefix = resourceName.prefix ? resourceName.prefix : "";
    const name = resourceName.name ? resourceName.name : "";
    const suffix = resourceName.suffix ? resourceName.suffix : "";
    const separator = resourceName.separator ? resourceName.separator : "";
    const separatorCount = prefix && suffix ? 2 : (prefix || suffix ? 1 : 0);
    const maxPrefixLength = maxLength - name.length - suffix.length - (separator.length * separatorCount);
    const prefixLength = prefix.length > maxPrefixLength ? maxPrefixLength : prefix.length;
    return {...resourceName, prefix: prefix?.substring(0, prefixLength)};
}

export function truncateName(resourceName: ResourceName, maxLength: number): ResourceName {
    const prefix = resourceName.prefix ? resourceName.prefix : "";
    const name = resourceName.name ? resourceName.name : "";
    const suffix = resourceName.suffix ? resourceName.suffix : "";
    const separator = resourceName.separator ? resourceName.separator : "";
    const separatorCount = prefix && suffix ? 2 : (prefix || suffix ? 1 : 0);
    const maxNameLength = maxLength - prefix.length - suffix.length - (separator.length * separatorCount);
    let nameLength = name.length > maxNameLength ? maxNameLength : name.length;
    return {...resourceName, name: name.substring(0, nameLength)};
}

export function truncateNameComponents(splitterPattern ?: RegExp): ShorteningStrategy {
    return function (resourceName: ResourceName, maxLength: number): ResourceName {
        const prefix = resourceName.prefix ? resourceName.prefix : "";
        const name = resourceName.name ? resourceName.name : "";
        const suffix = resourceName.suffix ? resourceName.suffix : "";
        const resourceComponentSeparator = resourceName.separator ? resourceName.separator : "";
        const separatorCount = prefix && suffix ? 2 : (prefix || suffix ? 1 : 0);
        const maxNameLength = maxLength - prefix.length - suffix.length - (resourceComponentSeparator.length * separatorCount);

        let separator = resourceComponentSeparator || "";
        const splitter = splitterPattern || separator;
        let nameComponents = splitterPattern || separator ? name?.split(splitter) : [name];
        if (!nameComponents || nameComponents.length == 1) {
            return {...resourceName};
        }

        let excessChars = name.length - maxNameLength;
        for (let i = nameComponents.length - 1; excessChars > 0 && (i != 0 || Math.max(...nameComponents.map(c => c.length)) > 1); --i) {
            if (i < 0) {
                i = nameComponents.length - 1;
            }

            const c = nameComponents[i];

            if (!c) {
                nameComponents.splice(i - 1, 2); // drop empty component (should only happen if the original name includes a sequence of multiple separators)
            } else if (c.length == 1) {
                continue; // don't remove the first character in a component
            } else {
                nameComponents[i] = c.substring(0, c.length - 1);
            }
            --excessChars;
        }
        const shortenedName = nameComponents.join("");
        if (shortenedName.length > maxNameLength) {
            throw new Error(`Minimum shortened length ${shortenedName.length} exceeds maximum length of ${maxNameLength}`);
        }
        return {...resourceName, name: shortenedName};
    }
}

export function truncateSuffix(resourceName: ResourceName, maxLength: number): ResourceName {
    const prefix = resourceName.prefix ? resourceName.prefix : "";
    const name = resourceName.name ? resourceName.name : "";
    const suffix = resourceName.suffix ? resourceName.suffix : "";
    const separator = resourceName.separator ? resourceName.separator : "";
    const separatorCount = prefix && suffix ? 2 : (prefix || suffix ? 1 : 0);
    const maxSuffixLength = maxLength - prefix.length - name.length - (separator.length * separatorCount);
    const suffixLength = suffix.length > maxSuffixLength ? maxSuffixLength : suffix.length;
    return {...resourceName, suffix: suffix.substring(0, suffixLength)};
}

export function hashNameSha256(resourceName: ResourceName, maxLength: number): ResourceName {
    const prefix = resourceName.prefix ? resourceName.prefix : "";
    const name = resourceName.name ? resourceName.name : "";
    const suffix = resourceName.suffix ? resourceName.suffix : "";
    const separator = resourceName.separator ? resourceName.separator : "";
    const separatorCount = prefix && suffix ? 2 : (prefix || suffix ? 1 : 0);
    const maxNameLength = maxLength - prefix.length - suffix.length - (separator.length * separatorCount);
    let nameLength = name.length > maxNameLength ? maxNameLength : name.length;

    const hash = crypto.createHash('sha256');
    hash.update(name);
    const digest = hash.digest('hex');
    return {...resourceName, name: digest.substring(0, nameLength)};
}

export function regexpMatch(description: string, pattern: RegExp, fix: FixFunc | undefined = undefined): SanitizationRule {
    return {
        description: description ? description : `The supplied string must match the following pattern: ${pattern.source}`,
        validate: (resourceName) => pattern.test(resourceName),
        fix
    }
}

export function regexpNotMatch(name: string, pattern: RegExp, fix: FixFunc | undefined = undefined): SanitizationRule {
    return {
        description: name,
        validate: (resourceName) => !pattern.test(resourceName),
        fix
    }
}