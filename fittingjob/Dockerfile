FROM python:3.8

WORKDIR /app

COPY ./Pipfile /app/Pipfile
COPY ./Pipfile.lock /app/Pipfile.lock
RUN pip install pipenv
# Install build dependencies first as fbprophet requires them during build
RUN pip install numpy cython pandas convertdate lunarcalendar
RUN pipenv install --system

COPY ./ /app/

ENTRYPOINT ["./train.py", "/fittingjob-config/config.json"]
