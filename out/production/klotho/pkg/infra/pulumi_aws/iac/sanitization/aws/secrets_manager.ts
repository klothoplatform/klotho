import { regexpMatch } from '../sanitizer'

export const secret = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 512,
            rules: [regexpMatch('', /^[\w/+=.@-]+$/, (n) => n.replace(/[^\w/+=.@-]/g, '_'))],
        }
    },
}
