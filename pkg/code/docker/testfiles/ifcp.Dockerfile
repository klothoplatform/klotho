FROM python:3.11-slim-bookworm

RUN mkdir /app
WORKDIR /app

RUN pip3 install awscli
ENV PYTHONPATH=.:${PYTHONPATH}
EXPOSE 80

COPY requirements.txt .

RUN pip3 install -r requirements.txt

RUN mkdir ./src
COPY ./src ./src

RUN mkdir binaries/

COPY container_start.sh .
RUN chmod +x container_start.sh

ENTRYPOINT [ "./container_start.sh" ]
