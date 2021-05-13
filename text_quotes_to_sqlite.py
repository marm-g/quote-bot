with open('quotes.txt', 'r') as f:
    data = f.readlines()

query = "INSERT INTO quote (quoteText, dateAdded) VALUES "
for line in data:
    line = line.replace('\n', '')
    line = line.replace('\'', '')
    query += "('{}', 0),".format(line)
query = query[:-1]
print(query)
