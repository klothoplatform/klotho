import {regexpMatch, SanitizationOptions} from "../sanitizer";

export default {
    privateDnsNamespace: {
        nameValidation(): SanitizationOptions {
            return {
                minLength: 1,
                maxLength: 253,
                rules: [
                    regexpMatch("", /^[!-~]+$/, (s) => s.replace(/[^!-~]/g, "_"))
                ]
            }
        }
    }
}