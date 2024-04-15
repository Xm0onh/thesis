import jinja2

template_text = """
############################
  ##########################
  ##   Lambda Functions {{ responder_id }} ##
  ##########################
  ResponderFunction{{ responder_id }}:
    Type: 'AWS::Lambda::Function'
    Properties:
      Handler: main
      Runtime: provided.al2023
      Role: !GetAtt ResponderLambdaExecutionRole.Arn
      Code:
        S3Bucket: thesisubc
        S3Key: responder.zip
      Environment:
        Variables:
          DDB_TABLE_NAME: droplets
          BLOCKCHAIN_S3_BUCKET: thesisubc
          RESPONDER_ID: {{ responder_id }}
          SETUP_DB: setup
      MemorySize: 3006
      Timeout: 900

  ###################
  ##   SNS Topic {{ responder_id }} ##
  ###################
  Responder{{ responder_id }}:
    Type: AWS::SNS::Topic
    Properties: 
      DisplayName: Responder{{ responder_id }}
      TopicName: Responder{{ responder_id }}Topic

  ###########################
  ##   SNS Subscriptions {{ responder_id }} ##
  ###########################

  Responder{{ responder_id }}SNSSubscription:
    Type: AWS::SNS::Subscription
    Properties:
      Protocol: lambda
      TopicArn: !Ref Responder{{ responder_id }}
      Endpoint: !GetAtt ResponderFunction{{ responder_id }}.Arn

  LambdaInvokePermission{{ responder_id }}:
    Type: AWS::Lambda::Permission
    Properties:
      Action: "lambda:InvokeFunction"
      FunctionName: !GetAtt ResponderFunction{{ responder_id }}.Arn
      Principal: "sns.amazonaws.com"
      SourceArn: !Ref Responder{{ responder_id }}
############################
"""

# Create a template from the string
template = jinja2.Template(template_text)


with open("responder_configurations.yaml", "w") as f:
    for i in range(6, 51):  
        f.write(template.render(responder_id=i))
        f.write("\n\n")  

print("YAML file has been created with the configurations.")