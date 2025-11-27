# Baofeng DM-32UV Roaming Setup Instructions

Based on the user manual and online resources, here is how to set up roaming on your Baofeng DM-32UV radio.

## Prerequisites

1. **CPS Software:** Ensure you have the Customer Programming Software (CPS) installed for the DM-32UV.
2. **Programming Cable:** Connect your radio to the computer.

## Step 1: Program Repeater Channels

Before creating a roaming zone, you must program the individual repeater channels. We have prepared `roam-channels.csv` with the Mi5 network data for this purpose.

For each channel, ensure the following are set:

* **RX Frequency:** The repeater's output frequency.
* **TX Frequency:** The repeater's input frequency.
* **Color Code:** Correct Color Code (usually 1 for Mi5).
* **Time Slot:** Correct Time Slot (usually 1 or 2).
* **Channel Name:** A descriptive name (e.g., "Detroit").

## Step 2: Create Roaming Zones

A "Roaming Zone" is a list of channels the radio will scan to find the best signal. We have prepared `roam-zones.csv` with the Mi5 zones.

1. In the CPS, navigate to **Roaming Zones**.
2. Create a new Zone (e.g., "Detroit Area").
3. Add the relevant channels (e.g., Detroit, Mt.Clemens, Novi, Southgate) to this zone.
4. Repeat for other areas (Flint/Saginaw, Grand Rapids, etc.).

## Step 3: Assign Roaming to a Channel

You need to link a channel to a roaming zone.

1. Go to the **Channel** settings for your main talk channel (e.g., "Mi5 Talk").
2. Find the **Roaming Zone** setting (sometimes called "Roam List" or similar).
3. Select the Roaming Zone you created in Step 2.
4. **Auto Roaming:** You can optionally set "Auto Roaming" to "On" to have it start automatically, or configure a timer.

## Step 4: Assign a Roaming Key (Optional)

You can manually trigger roaming using a side key.

1. In the CPS, go to **Button Definitions** (or "Key Function").
2. Assign **Roaming** (or "Roam On/Off") to a side key (e.g., SK1 or SK2).
3. Write the configuration to the radio.

## How to Use

* **Automatic:** If assigned to a channel, the radio should check for the strongest repeater in the zone when you switch to that channel or when the signal drops.
* **Manual:** Press the assigned Side Key to force the radio to scan the roaming list and lock onto the strongest signal.
