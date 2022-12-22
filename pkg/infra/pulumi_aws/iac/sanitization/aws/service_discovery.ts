import { regexpMatch, regexpNotMatch, SanitizationOptions } from '../sanitizer'

export const privateDnsNamespace = {
    nameValidation(): SanitizationOptions {
        return {
            minLength: 1,
            maxLength: 253,
            rules: [regexpMatch('', /^[!-~]+$/, (s) => s.replace(/[^!-~]/g, '_'))],
        }
    },
}

export const service = {
    nameValidation(): SanitizationOptions {
        return {
            minLength: 1,
            maxLength: 127,
            rules: [
                regexpMatch(
                    '',
                    /((?=^.{1,127}$)^([a-zA-Z0-9_][a-zA-Z0-9-_]{0,61}[a-zA-Z0-9_]|[a-zA-Z0-9])(\.([a-zA-Z0-9_][a-zA-Z0-9-_]{0,61}[a-zA-Z0-9_]|[a-zA-Z0-9]))*$)|(^\.$)/
                ),
                regexpMatch(
                    'Name can only contain alphanumeric characters, underscores, hyphens, and periods',
                    /^[\w.-]$/,
                    (s) => s.replace(/[^\w.-]/g, '_')
                ),
                regexpNotMatch('Name component must not start with a hyphen', /(^-)|(\.-)/, (s) =>
                    s.replace(/(^-+)|((?<=\.)-+)/g, '')
                ),
            ],
        }
    },
}
