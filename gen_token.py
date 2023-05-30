import importlib

def auto_install(package):
    try:
        importlib.import_module(package)
    except ImportError:
        import subprocess
        subprocess.call(["pip", "install", package])

try:
    import undetected_chromedriver as uc
    from selenium.webdriver.common.by import By
    import selenium.webdriver.support.expected_conditions as EC
    from selenium.webdriver.support.ui import WebDriverWait
    from bs4 import BeautifulSoup
    import time
except ImportError as e:
    auto_install(str(e).split("'")[1])
    import undetected_chromedriver as uc
    from selenium.webdriver.common.by import By
    import selenium.webdriver.support.expected_conditions as EC
    from selenium.webdriver.support.ui import WebDriverWait
    from bs4 import BeautifulSoup
    import time


def get_access_token(email: str, password: str):
    startT = time.time()
    driver = uc.Chrome(
        headless=True,
        version_main=108,
    )
    driver.get('https://chat.openai.com/auth/login')

    driver.find_element(
        By.XPATH, '//*[@id="__next"]/div[1]/div[1]/div[4]/button[1]').click()

    try:
        WebDriverWait(driver, 20).until(
            EC.visibility_of_element_located((By.ID, 'username')))
    except:
        return 'timeout while waiting for login page'
    finally:
        driver.find_element(By.ID, 'username').send_keys(email)

    driver.find_element(By.NAME, "action").click()
    try:
        WebDriverWait(driver, 20).until(
            EC.visibility_of_element_located((By.ID, 'password')))
    except:
        return 'timeout while waiting for password page'
    finally:
        driver.find_element(By.ID, 'password').send_keys(password)

    driver.find_element(By.NAME, "action").click()
    try:
        WebDriverWait(driver, 20).until(
            EC.url_to_be('https://chat.openai.com/'))
    except:
        return 'timeout/invalid credentials'
    finally:
        driver.get('https://chat.openai.com/api/auth/session')

    soup = BeautifulSoup(driver.page_source, 'html.parser')
    print(time.time() - startT)
    return soup.find('pre').text

print("Welcome to the OpenAI Access Token Generator")
print("[!] Make sure you have the latest version of Chrome installed (>108)")
print("Please enter your OpenAI email and password")

Email = input('Email: ')
Password = input('Password: ')

startTime = time.time()
token = get_access_token(Email, Password)

if token.startswith('timeout'):
    print(token)

else:
    import json
    json_data = json.loads(token)
    print(f"Your access token is: {json_data['accessToken']}")
    print(f"All done in {time.time() - startTime} seconds")

