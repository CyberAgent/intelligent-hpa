FROM python:3.9

WORKDIR /app

COPY ./Pipfile /app/Pipfile
COPY ./Pipfile.lock /app/Pipfile.lock
RUN python -c "import json; l=json.load(open('Pipfile.lock')); [print(f'{k}{v[\"version\"]}') for k,v in l['default'].items()]" > /tmp/reqs.txt && pip install -r /tmp/reqs.txt

COPY ./ /app/

ENTRYPOINT ["./train.py", "/fittingjob-config/config.json"]
