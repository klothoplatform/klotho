import { regexpMatch, regexpNotMatch, SanitizationOptions } from '../sanitizer'

export const bucket = {
    nameValidation(): SanitizationOptions {
        return {
            minLength: 3,
            maxLength: 63,
            rules: [
                regexpMatch(
                    'Bucket names can consist only of lowercase letters, numbers, dots (.), and hyphens (-).',
                    /^[a-z\d.-]+$/,
                    (n) => n.toLowerCase().replace(/[^a-z\d-.]+/g, '-')
                ),
                {
                    description: 'Bucket names must not start with the prefix "xn--".',
                    validate: (n) => !n.startsWith('xn--'),
                    fix: (n) => n.replace(/^xn--/, ''),
                },
                {
                    description: 'Bucket names must not end with the suffix "-s3alias".',
                    validate: (n) => !n.endsWith('-s3alias'),
                    fix: (n) => n.replace(/-s3alias$/, ''),
                },
                regexpMatch(
                    'Bucket names must begin and end with a letter or number.',
                    /^[a-z\d].+[a-z\d]$/,
                    (n) => n.replace(/^[^a-zA-Z\d]+/, '').replace(/[^a-zA-Z\d]+$/g, '')
                ),
                {
                    description: 'Bucket names must not contain two adjacent periods.',
                    validate: (n) => !n.includes('..'),
                    fix: (n) => n.replaceAll('..', '.'),
                },
                regexpNotMatch(
                    'Bucket names must not be formatted as an IP address.',
                    /^(?:\d{1,3}\.){3}\d{1,3}$/,
                    (n) => n.replaceAll('.', '-')
                ),
            ],
        }
    },
}
