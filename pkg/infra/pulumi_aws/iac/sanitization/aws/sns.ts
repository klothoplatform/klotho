import { regexpMatch } from '../sanitizer'

export const topic = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 256,
            rules: [regexpMatch('', /^[\w-]+$/, (s) => s.replace(/[^\w-]/g))],
        }
    },
}

export const fifoTopic = {
    nameValidation() {
        return {
            minLength: 6,
            maxLength: 256,
            rules: [
                regexpMatch(
                    "The FIFO topic name must be made up of only uppercase and lowercase ASCII letters, numbers, underscores, and hyphens, and must end with the '.fifo' suffix",
                    /^[\w-]{1,251}\.fifo$/,
                    (s) => {
                        if (!s.endsWith('.fifo')) {
                            s = s.substring(0, s.length - 6)
                        }
                        s = `${s.replace(/[^\w-]/g, '-')}.fifo`
                        return s
                    }
                ),
            ],
        }
    },
}
