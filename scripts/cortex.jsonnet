//local resource = import 'f58ddb33-b06f-45dd-b3cf-3483bc2032cc.json';

local map_url(name, action) =
    local parts = std.split(action.image, '/');
    local patch = {
    //only if first part of split by / has '.', then only replace docker registry url (assuming . means its a hostname or IP)
        "image": if std.length(parts) > 1 && std.length(std.findSubstr('.', parts[0])) > 1 then
            std.join('/', [std.extVar('DOCKER_PREGISTRY_URL')] + parts[1:])
        else
            std.join('/', parts)
    };
    std.mergePatch(action, patch)
;
std.mergePatch(resource, {
    "dependencies": {
        "actions": std.mapWithKey(map_url, resource.dependencies.actions)
    }
})
