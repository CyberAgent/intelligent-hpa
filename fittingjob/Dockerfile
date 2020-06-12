FROM python:3.8

WORKDIR /app

COPY ./Pipfile /app/Pipfile
COPY ./Pipfile.lock /app/Pipfile.lock
RUN pip install pipenv
RUN pipenv install --skip-lock --system

COPY ./ /app/

ENTRYPOINT ["./train.py", "/fittingjob-config/config.json"]
