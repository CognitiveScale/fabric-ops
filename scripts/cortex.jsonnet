//local resource = import 'f58ddb33-b06f-45dd-b3cf-3483bc2032cc.json';

local map_action(name, action) =
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

local map_skill(name, skill) =
    {
        "actions": if std.length(skill.actions) > 0 then [map_action(skill.actions[0].name, skill.actions[0])] else []
    }
;

local actions = if std.objectHas(resource.dependencies, 'actions') then resource.dependencies.actions else {};
local skills = if std.objectHas(resource.dependencies, 'skills') then resource.dependencies.skills else {};

local resource_mapped = std.mergePatch(resource, {"dependencies": {"actions": std.mapWithKey(map_action, actions)}});

std.mergePatch(resource_mapped, { "dependencies": { "skills": std.mapWithKey(map_skill, skills) } })
