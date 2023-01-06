import {regexpMatch} from "../sanitizer";

export const role = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 64,
            rules: [
                regexpMatch(
                    "",
                    /^[\w+=,.@-]+$/,
                    s => s.replace(/[^\w+=,.@-]/g)
                )
            ]
        }
    }
}

export const lambdaFunction = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 64,
            rules: [
                regexpMatch(
                    "",
                    /^[\w-]+$/,
                    s => s.replace(/[^\w-]/g)
                )
            ]
        }
    }
}