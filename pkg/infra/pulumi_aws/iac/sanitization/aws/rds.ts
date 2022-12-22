import { regexpMatch } from '../sanitizer'

export const dbSubnetGroup = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 255,
            rules: [
                regexpMatch('', /^[\w -.]+$/, (s) => s.replace(/[^\w -.]/g, '_')),
                regexpMatch('Name must start with a letter', /^[a-zA-Z]/, (s) =>
                    s.replace(/^[^a-zA-Z]+/, '')
                ),
                {
                    description: "Name must not be 'default'",
                    validate: (s) => s.toLowerCase() !== 'default',
                },
            ],
        }
    },
}

export const engine = {
    pg: {
        database: {
            nameValidation() {
                return {
                    minLength: 1,
                    maxLength: 63,
                    rules: [],
                }
            },
        },
    },
}

export const dbProxy = {
    nameValidation() {
        return {
            minLength: 1,
            rules: [
                regexpMatch('', /^[\da-zA-Z-]+$/, (s) => s.replace(/[^\da-zA-Z-]/g, '-')),
                regexpMatch('Identifier must start with a letter', /^[a-zA-Z]/, (s) =>
                    s.replace(/^[^a-zA-Z]+/, '')
                ),
                {
                    description: 'Identifier must not end with a hyphen',
                    validate: (s) => !s.endsWith('-'),
                    fix: (s) => s.replace(/-+$/, ''),
                },
                regexpMatch('Identifier must not contain consecutive hyphens', /--/, (s) =>
                    s.replace(/--+/g, '-')
                ),
            ],
        }
    },
}

export const instance = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 63,
            rules: [
                regexpMatch('', /^[\da-zA-Z-]+$/, (s) => s.replace(/[^\da-zA-Z-]/g, '-')),
                regexpMatch('Identifier must start with a letter', /^[a-zA-Z]/, (s) =>
                    s.replace(/^[^a-zA-Z]+/, '')
                ),
                {
                    description: 'Identifier must not end with a hyphen',
                    validate: (s) => !s.endsWith('-'),
                    fix: (s) => s.replace(/-+$/, ''),
                },
                regexpMatch('Identifier must not contain consecutive hyphens', /--/, (s) =>
                    s.replace(/--+/g, '-')
                ),
            ],
        }
    },
}
