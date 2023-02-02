import { regexpMatch } from '../sanitizer'

export const vpc = {
    securityGroup: {
        nameValidation() {
            return {
                minLength: 1,
                maxLength: 255,
                rules: [
                    regexpMatch(
                        'a-z, A-Z, 0-9, spaces, and ._-:/()#,@[]+=&;{}!$*',
                        /^[\w -.:/()#,@\[\]+=&;{}!$*]+$/,
                        (s) => s.replace(/[^\w -.:/()#,@\[\]+=&;{}!$*]/g, '_')
                    ),
                ],
            }
        },
    },
}

export const classic = {
    securityGroup: {
        nameValidation() {
            return {
                minLength: 1,
                maxLength: 255,
                rules: [
                    regexpMatch('Must only containASCII characters', /^[[:ascii:]]+$/, (s) =>
                        s.replace(/[^[:ascii:]]/g, '_')
                    ),
                    {
                        description: "Cannot start with 'sg-'",
                        validate: (s) => !s.startsWith('sg-'),
                        fix: (s) => s.substring(3),
                    },
                ],
            }
        },
    },
}
