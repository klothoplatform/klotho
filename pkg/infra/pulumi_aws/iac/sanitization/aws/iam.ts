import { regexpMatch } from '../sanitizer'

export const role = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 64,
            rules: [regexpMatch('', /^[\w+=,.@-]+$/, (s) => s.replace(/[^\w+=,.@-]/g, '_'))],
        }
    },
}

export const policy = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 128,
            rules: [regexpMatch('', /^[\w+=,.@-]+$/, (s) => s.replace(/[^\w+=,.@-]/g, '_'))],
        }
    },
}
