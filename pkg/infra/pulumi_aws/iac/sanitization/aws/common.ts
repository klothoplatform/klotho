import { regexpMatch } from '../sanitizer'

export const tag = {
    keyValidation() {
        return {
            minLength: 1,
            maxLength: 128,
            rules: [
                regexpMatch('', /^[\p{L}\p{Z}\p{N}_.:/=+\-@]+$/u, (n) =>
                    n.replace(/[\p{L}\p{Z}\p{N}_.:/=+\-@]/gu, '_')
                ),
            ],
        }
    },

    valueValidation() {
        return {
            minLength: 0,
            maxLength: 256,
            rules: [
                regexpMatch('', /^[\p{L}\p{Z}\p{N}_.:/=+\-@]+$/u, (n) =>
                    n.replace(/[\p{L}\p{Z}\p{N}_.:/=+\-@]/gu, '_')
                ),
                {
                    description:
                        "The aws: prefix is prohibited for tags; it's reserved for AWS use.",
                    apply: (v) => !v.startsWith('aws:'),
                    fix: (v) => v.replace(/^aws:/, ''),
                },
            ],
        }
    },
}
