FROM python:3.8

WORKDIR /app

COPY ./Pipfile /app/Pipfile
COPY ./Pipfile.lock /app/Pipfile.lock
RUN pip install pipenv
# Install all dependencies from Pipfile.lock except fbprophet first,
# then install fbprophet (which requires its deps during build)
RUN python -c "import json; l=json.load(open('Pipfile.lock')); [print(f'{k}{v[\"version\"]}') for k,v in l['default'].items() if k!='fbprophet']" > /tmp/reqs.txt && pip install -r /tmp/reqs.txt
RUN pip install fbprophet==0.7.1 --no-build-isolation

COPY ./ /app/

ENTRYPOINT ["./train.py", "/fittingjob-config/config.json"]
