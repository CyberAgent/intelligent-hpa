FROM python:3.8

WORKDIR /app

COPY ./Pipfile /app/Pipfile
COPY ./Pipfile.lock /app/Pipfile.lock
RUN pip install pipenv
# Install all dependencies except fbprophet first, then install fbprophet
# (fbprophet requires its dependencies during build)
RUN python -c "import json; f=open('Pipfile.lock'); l=json.load(f); f.close(); [print(k+v['version']) for k,v in l['default'].items() if k!='fbprophet']" > /tmp/reqs.txt && pip install -r /tmp/reqs.txt
RUN pip install fbprophet==0.7.1

COPY ./ /app/

ENTRYPOINT ["./train.py", "/fittingjob-config/config.json"]
