import os
import re
import botocore.session

TASK_DEFINITION_REGISTER_ARG_NAMES = [
  'containerDefinitions',
  'cpu',
  'executionRoleArn',
  'family',
  'ipcMode',
  'memory',
  'networkMode',
  'pidMode',
  'placementConstraints',
  'proxyConfiguration',
  'requiresCompatibilities',
  'tags',
  'taskRoleArn',
  'volumes',
]

session = botocore.session.get_session()
ecs = session.create_client('ecs')
ecr = session.create_client('ecr')

ECS_CLUSTER = os.environ.get('ECS_CLUSTER')
ECS_SERVICE = os.environ.get('ECS_SERVICE')
ECS_TASKDEF = os.environ.get('ECS_TASKDEF')
IMAGE_NAME = os.environ.get('IMAGE_NAME')

def handler(event, context):
    try:
        return deploy(event['version'])
    except Exception as e:
        return dict(
            status='ERROR',
            message=str(e)
        )

def deploy(service_version):
    task_def_res = ecs.describe_task_definition(
        taskDefinition=ECS_TASKDEF
    )
    task_def = task_def_res['taskDefinition']

    new_image = IMAGE_NAME + ':' + service_version
    main_container = task_def['containerDefinitions'][0]
    main_container['image'] = new_image

    log_confg = main_container['logConfiguration']
    if log_confg['logDriver'] == 'awslogs':
        log_stream_prefix = '%s/%s/' % (main_container['name'], service_version)
        log_confg['options']['awslogs-stream-prefix'] = log_stream_prefix

    task_def_req = {
        key: task_def[key]
        for key in TASK_DEFINITION_REGISTER_ARG_NAMES
        if key in task_def
    }
    register_res = ecs.register_task_definition(**task_def_req)
    new_task_def_arn = register_res['taskDefinition']['taskDefinitionArn']

    ecs.update_service(
        cluster=ECS_CLUSTER,
        service=ECS_SERVICE,
        taskDefinition=new_task_def_arn
    )

    return dict(
        status='OK',
        new_image=new_image,
        task_definition=new_task_def_arn
    )
