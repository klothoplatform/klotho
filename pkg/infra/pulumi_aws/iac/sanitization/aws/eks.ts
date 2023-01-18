import { regexpMatch } from '../sanitizer'

export const cluster = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 100,
            rules: [
                regexpMatch(
                    'The name can contain only alphanumeric characters (case-sensitive) and hyphens.',
                    /^[a-zA-Z\d-]+$/,
                    (s) => s.replace(/[^a-zA-Z\d-]/g, '_')
                ),
                regexpMatch('The name must start with an alphabetic character', /^[a-zA-Z]/, (s) =>
                    s.replace(/^[^a-zA-Z]+/, '')
                ),
            ],
        }
    },
}
