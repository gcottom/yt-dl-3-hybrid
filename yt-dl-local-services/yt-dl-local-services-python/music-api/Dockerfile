FROM python:3.10.14
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY music-api.py .
ENTRYPOINT ["python", "music-api.py"]
