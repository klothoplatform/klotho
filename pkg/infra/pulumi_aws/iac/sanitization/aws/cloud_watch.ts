import { regexpMatch } from '../sanitizer'

export const logGroup = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 512,
            rules: [
                regexpMatch(
                    "Log group names consist of the following characters: a-z, A-Z, 0-9, '_' (underscore), '-' (hyphen), '/' (forward slash), '.' (period), and '#' (number sign).",
                    /^[-._/#A-Za-z\d]+$/,
                    (n) => n.replace(/[^-._/#A-Za-z\d]/g, '_')
                ),
            ],
        }
    },
}
