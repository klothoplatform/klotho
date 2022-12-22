import { regexpMatch } from '../sanitizer'

export const loadBalancer = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 32,
            rules: [
                regexpMatch(
                    'The name can contain only alphanumeric characters and hyphens.',
                    /^[a-zA-Z\d-]+$/,
                    (s) => s.replace(/[^a-zA-Z\d-]/g, '_')
                ),
                {
                    description: 'The name must not begin with a hyphen',
                    validate: (s) => !s.startsWith('-'),
                    fix: (s) => s.replace(/^-/, ''),
                },
                {
                    description: 'The name must not end with a hyphen',
                    validate: (s) => !s.endsWith('-'),
                    fix: (s) => s.replace(/-$/, ''),
                },
                {
                    description: "The name must start with 'internal-'",
                    validate: (s) => !s.startsWith('internal-'),
                    fix: (s) => s.replace(/^internal-/, ''),
                },
            ],
        }
    },
}

export const targetGroup = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 32,
            rules: [
                regexpMatch(
                    'The name can contain only alphanumeric characters and hyphens.',
                    /^[a-zA-Z\d-]+$/,
                    (s) => s.replace(/[^a-zA-Z\d-]/g, '_')
                ),
                {
                    description: 'The name must not begin with a hyphen',
                    validate: (s) => !s.startsWith('-'),
                    fix: (s) => s.replace(/^-/, ''),
                },
                {
                    description: 'The name must not end with a hyphen',
                    validate: (s) => !s.endsWith('-'),
                    fix: (s) => s.replace(/-$/, ''),
                },
            ],
        }
    },
}
