import plotly.express as px
import plotly.io as pio
import sqlite3
import pandas as pd

def format_currency(value):
    gold = value // 10000
    silver = (value % 10000) // 100
    copper = value % 100
    return f"{gold}g {silver}s {copper}c"

def craft_query(item_id, faction_id, realm_id):
    return f"""SELECT 
    strftime('%Y-%m-%d %H:00:00', timestamp, 'unixepoch') AS date_hour, 
    MIN(buyout) AS min_buyout,
    SUM(quantity) AS total_quantity
    FROM Auctions
    WHERE item_id = {item_id}
    AND faction_id = {faction_id}
    AND connected_realm_id = {realm_id}
    AND timestamp >= strftime('%s', 'now', '-7 days')
    AND buyout > 0
    GROUP BY strftime('%Y-%m-%d %H:00:00', timestamp, 'unixepoch')
    ORDER BY strftime('%Y-%m-%d %H:00:00', timestamp, 'unixepoch');
    """

def panda_query(query):
    # Load the query result into a Pandas DataFrame
    df = pd.read_sql_query(query, conn)
    # Convert the 'date_hour' column to a datetime object
    df['date_hour'] = pd.to_datetime(df['date_hour'])
    # Add a column for the day of the week
    df['day_of_week'] = df['date_hour'].dt.day_name()

    # Apply the currency formatting for the hover data
    df['formatted_min_buyout'] = df['min_buyout'].apply(format_currency)

    # Create the hover text using the formatted buyout data and volume
    df['hover_text'] = df['day_of_week'] + ", " + df['date_hour'].dt.strftime('%Y-%m-%d %H:%M') + \
                    "<br>Price: " + df['formatted_min_buyout'] + \
                    "<br>Volume: " + df['total_quantity'].astype(str)

    return df

def plot_data(df, item_name):
    # Create a Plotly figure
    fig = px.line(df, x='date_hour', y='min_buyout', 
                title=f'Min Buyout for {item_name}',
                labels={'min_buyout': 'Minimum Buyout Price', 'date_hour': 'Date'},
                custom_data=['hover_text'],
                name="Price history")

    # Update trace for custom hover template
    fig.update_traces(
        hovertemplate='%{customdata[0]}<extra></extra>'  # Use custom data in the hover template
    )

    # Add the volume as a bar chart with a secondary y-axis
    fig.add_bar(x=df['date_hour'], y=df['total_quantity'], name='Volume', yaxis='y2')

    # Update the layout to support a secondary y-axis
    fig.update_layout(
        yaxis2=dict(
            title='Volume',
            overlaying='y',
            side='right'
        )
    )

    # Optionally, customize the x-axis labels to include the day of the week
    fig.update_xaxes(tickformat='%a %Y-%m-%d %H:%M')

    # Set custom y-axis labels
    y_values = list(range(df['min_buyout'].min(), df['min_buyout'].max() + 10000, 10000))
    y_labels = [format_currency(val) for val in y_values]
    fig.update_yaxes(tickvals=y_values, ticktext=y_labels, secondary_y=False)

    # Set custom y-axis labels for volume
    # You can adjust the range and step for volume as per your data
    fig.update_yaxes(title_text='Volume', secondary_y=True)

    pio.write_image(fig, f"{item_name}_7_day_history.png", width=1920, height=1080)
    fig.show()

if __name__ == '__main__':
    # Connect to your SQLite database
    conn = sqlite3.connect('../data/db/blackwater.db')

    item_id = 15993 # mongoose
    faction_id = 0 # ally
    realm_id = 5284 # EU mirage race way

    # SQL query to fetch the item name
    item_query = f"SELECT name FROM Items WHERE item_id = {item_id};"

    # Execute the query
    cursor = conn.cursor()
    cursor.execute(item_query)
    item_name = cursor.fetchone()[0]  # This fetches the first (and only) result

    # Query AH data for an item
    query = craft_query(item_id, faction_id, realm_id)

    df = panda_query(query)

    plot_data(df, item_name)

    # Close the database connection
    conn.close()