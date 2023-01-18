import { regexpMatch } from '../sanitizer'

export const rule = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 64,
            rules: [regexpMatch('', /^[\w-.]+$/, (n) => n.replace(/[^\w-.]/g, '_'))],
        }
    },
}
