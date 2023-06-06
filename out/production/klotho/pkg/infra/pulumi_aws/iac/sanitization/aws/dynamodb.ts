import { regexpMatch } from '../sanitizer'

export const table = {
    nameValidation() {
        return {
            minLength: 3,
            maxLength: 255,
            rules: [
                regexpMatch('', /^[a-zA-Z0-9_.-]+$/, (n) => n.replace(/[^a-zA-Z0-9_.-]/g, '_')),
            ],
        }
    },
}
