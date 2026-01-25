FROM python:3.10-slim

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
ENV PIP_DISABLE_PIP_VERSION_CHECK=1

COPY requirements.txt /app/requirements.txt
COPY constraints.txt /app/constraints.txt
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --prefer-binary -r /app/requirements.txt -c /app/constraints.txt

COPY main.py /app/main.py

EXPOSE 9000

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "9000"]
