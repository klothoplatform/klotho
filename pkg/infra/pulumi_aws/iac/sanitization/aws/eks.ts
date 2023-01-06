import {regexpMatch} from "../sanitizer";

export const cluster = {
    nameValidation() {
        return {
            minLength: 1,
            maxLength: 100,
            rules: [
                regexpMatch("The name can contain only alphanumeric characters (case-sensitive) and hyphens.",
                    /^[a-zA-Z\d-]+$/,
                    s => s.replace(/[^a-zA-Z\d-]/g, "_")),
                {
                    description: "The name must start with an alphabetic character",
                    validate: s => !s.startsWith("-"),
                    fix: s => s.replace(/^-+/, "")
                }
            ]
        }
    }
}