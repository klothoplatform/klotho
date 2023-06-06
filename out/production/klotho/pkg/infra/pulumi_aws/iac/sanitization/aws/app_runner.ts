import { regexpMatch } from '../sanitizer'

export const service = {
    nameValidation() {
        return {
            minLength: 4,
            maxLength: 40,
            rules: [regexpMatch('', /^[\w-]+$/, (n) => n.replace(/[^\w-]/g, '-'))],
        }
    },
}
