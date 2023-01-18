import { regexpMatch } from '../sanitizer'

export const cluster = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 255,
            rules: [regexpMatch('', /^[\w-]+$/, (s) => s.replace(/[^\w-]/g, '_'))],
        }
    },
}

export const taskDefinition = {
    familyValidation() {
        return {
            minLength: 1,
            maxLength: 255,
            rules: [regexpMatch('', /^[\w-]+$/, (s) => s.replace(/[^\w-]/g, '_'))],
        }
    },
}

export const service = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 255,
            rules: [regexpMatch('', /^[\w-]+$/, (s) => s.replace(/[^\w-]/g, '_'))],
        }
    },
}
