import {regexpMatch} from "../sanitizer";

export default {
    roleNameValidation() {
        return {
            minLength: 1,
            maxLength: 64,
            rules: [
                regexpMatch(
                    "",
                    /[\w+=,.@-]+/,
                    s => s.replace(/[^\w+=,.@-]/g)
                )
            ]
        }
    }
}