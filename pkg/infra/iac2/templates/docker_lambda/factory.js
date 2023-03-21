"use strict";
exports.__esModule = true;
var aws = require("@pulumi/aws");
// noinspection JSUnusedLocalSymbols
function create(args) {
    return new aws.lambda.Function(args.ExecUnitName, {
        packageType: 'Image',
        imageUri: "TODO-image-uri",
        role: "TODO-role",
        name: "TODO-lambda-name",
        tags: {
            env: 'production',
            service: args.ExecUnitName
        }
    }, {
        dependsOn: [args.CloudwatchGroup]
    });
}
